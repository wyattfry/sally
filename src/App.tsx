import { useEffect, useRef, useState } from "react";
import { SallyPanel } from "./components/SallyPanel";
import { ScheduleViewer } from "./components/ScheduleViewer";
import { SpecButton } from "./components/SpecButton";
import { capturePage } from "./lib/capturePage";
import { extractScheduleItem, shouldAllowMockFallback, shouldFallbackToMock } from "./lib/extractApi";
import {
  getMothershipScheduleUrl,
  listMothershipProjects,
  listMothershipSchedules,
  saveMothershipScheduleItem
} from "./lib/mothershipApi";
import { mockExtractScheduleItem } from "./lib/mockExtraction";
import {
  getActiveMothershipContext,
  getProjectName,
  listScheduleItems,
  listZones,
  removeScheduleItem,
  saveActiveMothershipContext,
  saveProjectName,
  saveScheduleItem,
  saveZone
} from "./lib/storage";
import type { ActiveMothershipContext, MothershipProject, MothershipSchedule, ScheduleItem } from "./lib/types";

type PanelState =
  | { kind: "closed" }
  | { kind: "thinking"; tokenCount: number }
  | { kind: "review"; draft: ScheduleItem }
  | { kind: "minimized"; draft: ScheduleItem }
  | { kind: "error"; message: string };

const DEFAULT_CATEGORIES = [
  "Plumbing Fixture",
  "Lighting",
  "Appliance",
  "Hardware",
  "Finish",
  "Furniture",
  "Accessory"
];

export default function App() {
  const [panel, setPanel] = useState<PanelState>({ kind: "closed" });
  const [isScheduleOpen, setIsScheduleOpen] = useState(false);
  const [projectName, setProjectName] = useState("My New Project");
  const [scheduleItems, setScheduleItems] = useState<ScheduleItem[]>([]);
  const [zones, setZones] = useState<string[]>([]);
  const [mothershipProjects, setMothershipProjects] = useState<MothershipProject[]>([]);
  const [mothershipSchedules, setMothershipSchedules] = useState<MothershipSchedule[]>([]);
  const [activeMothershipContext, setActiveMothershipContext] =
    useState<ActiveMothershipContext | null>(null);
  const [toast, setToast] = useState<string | null>(null);
  const toastTimeoutRef = useRef<number | null>(null);

  useEffect(() => {
    refreshItemCount();
    refreshProjectName();
    refreshZones();
    refreshMothershipContext();
  }, []);

  useEffect(() => {
    if (!globalThis.chrome?.runtime?.onMessage) {
      return;
    }

    function handleRuntimeMessage(message: unknown) {
      if (isSpecMessage(message)) {
        handleSpecClick();
      } else if (isViewMessage(message)) {
        setPanel({ kind: "closed" });
        setIsScheduleOpen(true);
      }
    }

    chrome.runtime.onMessage.addListener(handleRuntimeMessage);
    return () => chrome.runtime.onMessage.removeListener(handleRuntimeMessage);
  }, []);

  useEffect(() => {
    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape" && panel.kind === "review") {
        setPanel({ kind: "minimized", draft: panel.draft });
      }
    }

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [panel]);

  useEffect(
    () => () => {
      if (toastTimeoutRef.current !== null) {
        window.clearTimeout(toastTimeoutRef.current);
      }
    },
    []
  );

  async function refreshItemCount() {
    const items = await listScheduleItems();
    setScheduleItems(items);
  }

  async function refreshZones() {
    setZones(await listZones());
  }

  async function refreshProjectName() {
    setProjectName(await getProjectName());
  }

  async function refreshMothershipContext() {
    try {
      const [projects, storedContext] = await Promise.all([
        listMothershipProjects(),
        getActiveMothershipContext()
      ]);
      setMothershipProjects(projects);
      const project =
        projects.find((candidate) => candidate.id === storedContext?.projectId) ?? projects[0];
      if (!project) {
        setMothershipSchedules([]);
        setActiveMothershipContext(null);
        return;
      }

      const schedules = await listMothershipSchedules(project.id);
      setMothershipSchedules(schedules);
      const schedule =
        schedules.find((candidate) => candidate.id === storedContext?.scheduleId) ?? schedules[0];
      if (!schedule) {
        setActiveMothershipContext(null);
        return;
      }

      const context = { projectId: project.id, scheduleId: schedule.id };
      setActiveMothershipContext(context);
      await saveActiveMothershipContext(context);
    } catch {
      setMothershipProjects([]);
      setMothershipSchedules([]);
      setActiveMothershipContext(null);
    }
  }

  function handleSpecClick() {
    setPanel({ kind: "thinking", tokenCount: 0 });
    window.setTimeout(async () => {
      const captured = capturePage(document, window.location);
      try {
        const proposal = await extractScheduleItem({
          capturedPage: captured,
          projectName,
          knownZones: zones,
          knownCategories: DEFAULT_CATEGORIES,
          onProgress: (tokenCount) => {
            setPanel((prev) => prev.kind === "thinking" ? { kind: "thinking", tokenCount } : prev);
          }
        });
        setPanel({ kind: "review", draft: proposal });
      } catch (error) {
        const message = error instanceof Error ? error.message : "Could not extract item.";
        if (shouldAllowMockFallback() && shouldFallbackToMock(error)) {
          try {
            const proposal = mockExtractScheduleItem(captured);
            setPanel({ kind: "review", draft: proposal });
            showToast("Using local mock fallback.");
            return;
          } catch {
            // fall through to visible error state
          }
        }

        setPanel({ kind: "error", message });
        showToast(message);
      }
    }, 250);
  }

  async function handleAddZone(zone: string) {
    const nextZones = await saveZone(zone);
    setZones(nextZones);
    if (panel.kind === "review") {
      setPanel({ kind: "review", draft: { ...panel.draft, zone: zone.trim() } });
    }
  }

  async function handleAccept(item: ScheduleItem) {
    if (activeMothershipContext?.scheduleId) {
      try {
        await saveMothershipScheduleItem(activeMothershipContext.scheduleId, item);
      } catch (error) {
        const message = error instanceof Error ? error.message : "Could not save item.";
        showToast(message);
        return;
      }
    }
    await saveScheduleItem(item);
    await refreshItemCount();
    setPanel({ kind: "closed" });
    showToast("Item added");
  }

  async function handleSelectMothershipProject(projectId: string) {
    const schedules = await listMothershipSchedules(projectId);
    setMothershipSchedules(schedules);
    const schedule = schedules[0];
    const context = schedule ? { projectId, scheduleId: schedule.id } : null;
    setActiveMothershipContext(context);
    if (context) {
      await saveActiveMothershipContext(context);
    }
  }

  async function handleSelectMothershipSchedule(scheduleId: string) {
    if (!activeMothershipContext) {
      return;
    }
    const context = { ...activeMothershipContext, scheduleId };
    setActiveMothershipContext(context);
    await saveActiveMothershipContext(context);
  }

  async function handleRenameProject(nextProjectName: string) {
    setProjectName(await saveProjectName(nextProjectName));
  }

  async function handleRemoveItem(itemId: string) {
    const items = await removeScheduleItem(itemId);
    setScheduleItems(items);
  }

  function handleViewItems() {
    if (activeMothershipContext) {
      window.open(
        getMothershipScheduleUrl(
          activeMothershipContext.projectId,
          activeMothershipContext.scheduleId
        ),
        "_blank"
      );
      setPanel({ kind: "closed" });
      return;
    }
    setPanel({ kind: "closed" });
    setIsScheduleOpen(true);
  }

  function showToast(message: string) {
    if (toastTimeoutRef.current !== null) {
      window.clearTimeout(toastTimeoutRef.current);
    }
    setToast(message);
    toastTimeoutRef.current = window.setTimeout(() => {
      setToast(null);
      toastTimeoutRef.current = null;
    }, 3200);
  }

  return (
    <div className="sally-root">
      <SpecButton
        onClick={handleSpecClick}
        itemCount={scheduleItems.length}
        onViewItems={handleViewItems}
      />
      {toast ? (
        <div className="toast" role="status">
          <span>{toast}</span>
          <div className="toast-progress" />
        </div>
      ) : null}
      {isScheduleOpen ? (
        <ScheduleViewer
          items={scheduleItems}
          projectName={projectName}
          onClose={() => setIsScheduleOpen(false)}
          onRemoveItem={handleRemoveItem}
          onRenameProject={handleRenameProject}
        />
      ) : null}
      {panel.kind === "minimized" ? (
        <button
          className="restore-draft-button"
          type="button"
          onClick={() => setPanel({ kind: "review", draft: panel.draft })}
        >
          Restore Sally draft
        </button>
      ) : null}
      {panel.kind === "error" ? (
        <div aria-label="Extraction error" className="sally-error-state">
          <p>{panel.message}</p>
          <div>
            <button type="button" onClick={handleSpecClick}>
              Retry extraction
            </button>
            <button type="button" onClick={() => setPanel({ kind: "closed" })}>
              Dismiss
            </button>
          </div>
        </div>
      ) : null}
      {panel.kind === "thinking" || panel.kind === "review" ? (
        <SallyPanel
          panel={panel}
          projectName={projectName}
          mothershipProjects={mothershipProjects}
          mothershipSchedules={mothershipSchedules}
          activeMothershipContext={activeMothershipContext}
          zones={zones}
          onCancel={() => setPanel({ kind: "closed" })}
          onChange={(draft) =>
            panel.kind === "review" ? setPanel({ ...panel, draft }) : undefined
          }
          onAddZone={handleAddZone}
          onSelectMothershipProject={handleSelectMothershipProject}
          onSelectMothershipSchedule={handleSelectMothershipSchedule}
          onAccept={handleAccept}
          onViewItems={handleViewItems}
        />
      ) : null}
    </div>
  );
}

function isSpecMessage(message: unknown): message is { type: string } {
  return (
    typeof message === "object" &&
    message !== null &&
    "type" in message &&
    message.type === "SALLY_SPEC_THIS_PAGE"
  );
}

function isViewMessage(message: unknown): message is { type: string } {
  return (
    typeof message === "object" &&
    message !== null &&
    "type" in message &&
    message.type === "SALLY_VIEW_ITEMS"
  );
}
