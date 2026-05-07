import { useEffect, useState } from "react";
import { SallyPanel } from "./components/SallyPanel";
import { SpecButton } from "./components/SpecButton";
import { capturePage } from "./lib/capturePage";
import { extractScheduleItem, shouldAllowMockFallback, shouldFallbackToMock } from "./lib/extractApi";
import {
  checkAuth,
  getSignInUrl,
  getMothershipScheduleUrl,
  listMothershipProjects,
  createMothershipProject,
  listMothershipSchedules,
  createMothershipSchedule,
  listMothershipScheduleColumns,
  getMothershipScheduleNextCode,
  saveMothershipScheduleItem
} from "./lib/mothershipApi";
import { mockExtractScheduleItem } from "./lib/mockExtraction";
import {
  getActiveContext,
  saveActiveContext
} from "./lib/storage";
import type { ActiveContext, Project, Schedule, ScheduleColumn, ScheduleItem } from "./lib/types";

type PanelState =
  | { kind: "closed" }
  | { kind: "signed-out" }
  | { kind: "signing-in" }
  | { kind: "thinking"; tokenCount: number }
  | { kind: "review"; draft: ScheduleItem; suggestedNewScheduleName?: string }
  | { kind: "minimized"; draft: ScheduleItem }
  | { kind: "added"; projectId: string; scheduleId: string; scheduleName: string }
  | { kind: "error"; message: string };

const SUGGESTED_SCHEDULE_NAMES = [
  "Appliance Schedule",
  "Cabinet Pulls",
  "Door Hardware Schedule",
  "Door Schedule",
  "Electrical Device Schedule",
  "Electrical Fixture Schedule",
  "Insulation Schedule",
  "Miscellaneous Devices",
  "Paint Schedule",
  "Specialties",
  "Window Schedule"
];

export default function App() {
  const [panel, setPanel] = useState<PanelState>({ kind: "closed" });
  const [projects, setProjects] = useState<Project[]>([]);
  const [schedules, setSchedules] = useState<Schedule[]>([]);
  const [columns, setColumns] = useState<ScheduleColumn[]>([]);
  const [zones, setZones] = useState<string[]>([]);
  const [activeContext, setActiveContext] = useState<ActiveContext | null>(null);

  useEffect(() => {
    refreshContext();
  }, []);

  useEffect(() => {
    if (!globalThis.chrome?.runtime?.onMessage) {
      return;
    }

    function handleRuntimeMessage(message: unknown) {
      if (isSpecMessage(message)) {
        handleSpecClick();
      } else if (isViewMessage(message)) {
        handleViewItems();
      }
    }

    chrome.runtime.onMessage.addListener(handleRuntimeMessage);
    return () => chrome.runtime.onMessage.removeListener(handleRuntimeMessage);
  }, [activeContext]);

  useEffect(() => {
    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape" && panel.kind === "review") {
        setPanel({ kind: "minimized", draft: panel.draft });
      }
    }

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [panel]);

  async function refreshContext() {
    try {
      const [fetchedProjects, storedContext] = await Promise.all([
        listMothershipProjects(),
        getActiveContext()
      ]);
      setProjects(fetchedProjects);

      const project = fetchedProjects[0];
      if (!project) {
        setSchedules([]);
        setColumns([]);
        setActiveContext(null);
        return;
      }

      const fetchedSchedules = await listMothershipSchedules(project.id);
      setSchedules(fetchedSchedules);
      const restoredSchedule = storedContext?.projectId === project.id
        ? fetchedSchedules.find((s) => s.id === storedContext.scheduleId)
        : undefined;
      const schedule = restoredSchedule ?? fetchedSchedules[0];
      if (!schedule) {
        setColumns([]);
        setActiveContext({ projectId: project.id, scheduleId: "" });
        return;
      }

      const [fetchedColumns] = await Promise.all([
        listMothershipScheduleColumns(schedule.id)
      ]);
      setColumns(fetchedColumns);

      const context = { projectId: project.id, scheduleId: schedule.id };
      setActiveContext(context);
      await saveActiveContext(context);
    } catch {
      setProjects([]);
      setSchedules([]);
      setColumns([]);
      setActiveContext(null);
    }
  }

  async function handleSpecClick() {
    const ok = await checkAuth();
    if (!ok) {
      setPanel({ kind: "signed-out" });
      return;
    }
    setPanel({ kind: "thinking", tokenCount: 0 });
    refreshContext();
    window.setTimeout(async () => {
      const captured = capturePage(document, window.location);
      const knownScheduleNames = [
        ...SUGGESTED_SCHEDULE_NAMES,
        ...schedules.map((s) => s.name).filter((n) => !SUGGESTED_SCHEDULE_NAMES.includes(n))
      ];
      try {
        const extracted = await extractScheduleItem({
          capturedPage: captured,
          knownCategories: [],
          knownScheduleNames,
          columns,
          scheduleId: activeContext?.scheduleId,
          onProgress: (tokenCount) => {
            setPanel((prev) => prev.kind === "thinking" ? { kind: "thinking", tokenCount } : prev);
          }
        });
        const { suggestedScheduleName } = extracted;
        let { item } = extracted;

        if (!item.data.code) {
          const nameFallback = suggestedScheduleName
            || schedules.find(s => s.id === activeContext?.scheduleId)?.name;
          if (nameFallback) {
            item = { ...item, data: { ...item.data, code: codePrefix(nameFallback) + "-1" } };
          }
        }

        const matchingSchedule = suggestedScheduleName
          ? schedules.find((s) => s.name.toLowerCase() === suggestedScheduleName.toLowerCase())
          : undefined;
        if (matchingSchedule && matchingSchedule.id !== activeContext?.scheduleId) {
          await handleSelectSchedule(matchingSchedule.id);
          item = { ...item, data: { ...item.data, code: codePrefix(matchingSchedule.name) + "-1" } };
        }
        setZones(extracted.knownZones);
        setPanel({
          kind: "review",
          draft: item,
          suggestedNewScheduleName:
            suggestedScheduleName && !matchingSchedule ? suggestedScheduleName : undefined
        });
      } catch (error) {
        const message = error instanceof Error ? error.message : "Could not extract item.";
        if (shouldAllowMockFallback() && shouldFallbackToMock(error)) {
          try {
            const item = mockExtractScheduleItem(captured);
            setPanel({ kind: "review", draft: item });
            return;
          } catch {
            // fall through to visible error state
          }
        }

        setPanel({ kind: "error", message });
      }
    }, 250);
  }

  async function handleAccept(item: ScheduleItem) {
    if (!activeContext?.scheduleId) {
      setPanel({ kind: "error", message: "Please select a project and schedule first." });
      return;
    }

    try {
      await saveMothershipScheduleItem(activeContext.scheduleId, item);
      const scheduleName = schedules.find((s) => s.id === activeContext.scheduleId)?.name ?? "schedule";
      setPanel({ kind: "added", projectId: activeContext.projectId, scheduleId: activeContext.scheduleId, scheduleName });
    } catch (error) {
      const message = error instanceof Error ? error.message : "Could not save item.";
      setPanel({ kind: "error", message });
    }
  }

  async function handleSelectProject(projectId: string) {
    if (projectId === "__add_new__") return;
    try {
      const fetchedSchedules = await listMothershipSchedules(projectId);
      setSchedules(fetchedSchedules);
      const schedule = fetchedSchedules[0];
      const fetchedColumns = schedule
        ? await listMothershipScheduleColumns(schedule.id)
        : [];
      setColumns(fetchedColumns);
      const context = { projectId, scheduleId: schedule?.id ?? "" };
      setActiveContext(context);
      await saveActiveContext(context);
    } catch {
      // selection failure is non-fatal; context stays as-is
    }
  }

  async function handleSelectSchedule(scheduleId: string) {
    if (!activeContext || scheduleId === "__add_new__") return;
    try {
      const [fetchedColumns, nextCode] = await Promise.all([
        listMothershipScheduleColumns(scheduleId),
        getMothershipScheduleNextCode(scheduleId),
      ]);
      setColumns(fetchedColumns);
      setPanel((prev) => {
        if (prev.kind !== "review") return prev;
        return { ...prev, draft: { ...prev.draft, data: { ...prev.draft.data, code: nextCode } } };
      });
    } catch {
      setColumns([]);
    }
    const context = { ...activeContext, scheduleId };
    setActiveContext(context);
    await saveActiveContext(context);
  }

  async function handleCreateProject(name: string): Promise<string | null> {
    try {
      const project = await createMothershipProject(name);
      setProjects((prev) => [...prev, project]);
      await handleSelectProject(project.id);
      return null;
    } catch (error) {
      return error instanceof Error ? error.message : "Could not create project.";
    }
  }

  async function handleCreateSchedule(name: string): Promise<string | null> {
    if (!activeContext?.projectId) {
      return "Select a project first.";
    }
    try {
      const schedule = await createMothershipSchedule(activeContext.projectId, name);
      setSchedules((prev) => [...prev, schedule]);
      await handleSelectSchedule(schedule.id);
      return null;
    } catch (error) {
      return error instanceof Error ? error.message : "Could not create schedule.";
    }
  }

  function handleSignIn() {
    const popup = window.open(
      getSignInUrl(),
      "sally-signin",
      "width=500,height=650,scrollbars=yes,resizable=yes"
    );
    if (!popup) return;
    setPanel({ kind: "signing-in" });

    const maxAttempts = 200; // ~5 minutes
    let attempts = 0;

    const interval = setInterval(async () => {
      attempts++;

      const ok = await checkAuth();
      if (ok) {
        clearInterval(interval);
        try { popup.close(); } catch { /* COOP may block */ }
        handleSpecClick();
        return;
      }

      let closed = false;
      try { closed = popup.closed; } catch { /* COOP may block */ }
      if (closed || attempts >= maxAttempts) {
        clearInterval(interval);
        setPanel({ kind: "signed-out" });
      }
    }, 1500);
  }

  function handleViewItems() {
    if (activeContext?.projectId && activeContext?.scheduleId) {
      window.open(
        getMothershipScheduleUrl(
          activeContext.projectId,
          activeContext.scheduleId
        ),
        "_blank"
      );
    }
    setPanel({ kind: "closed" });
  }

  return (
    <div className="sally-root">
      <SpecButton
        onClick={handleSpecClick}
        itemCount={0}
        onViewItems={handleViewItems}
      />
      {panel.kind === "minimized" ? (
        <button
          className="restore-draft-button"
          type="button"
          onClick={() => setPanel({ kind: "review", draft: panel.draft })}
        >
          Restore Sally draft
        </button>
      ) : null}
      {panel.kind === "signed-out" || panel.kind === "signing-in" ? (
        <aside className="sally-panel" aria-label="Sally">
          <div className="panel-header">
            <div className="panel-title">Sally</div>
          </div>
          <div className="panel-body">
            {panel.kind === "signing-in" ? (
              <p>Waiting for sign-in in the popup&hellip;</p>
            ) : (
              <>
                <p>Sign in to save items to your Sally projects.</p>
                <button className="action-button primary" type="button" onClick={handleSignIn}>
                  Sign in with Google
                </button>
              </>
            )}
          </div>
          <div className="panel-actions">
            <button className="action-button secondary" type="button" onClick={() => setPanel({ kind: "closed" })}>
              Cancel
            </button>
          </div>
        </aside>
      ) : panel.kind === "added" ? (
        <AddedConfirmation
          scheduleName={panel.scheduleName}
          projectUrl={getMothershipScheduleUrl(panel.projectId, panel.scheduleId)}
          onDismiss={() => setPanel({ kind: "closed" })}
        />
      ) : panel.kind !== "closed" && panel.kind !== "minimized" ? (
        <SallyPanel
          panel={panel}
          projects={projects}
          schedules={schedules}
          columns={columns}
          zones={zones}
          activeContext={activeContext}
          suggestedNewScheduleName={panel.kind === "review" ? panel.suggestedNewScheduleName : undefined}
          onCancel={() => setPanel({ kind: "closed" })}
          onChange={(draft) =>
            panel.kind === "review" ? setPanel({ ...panel, draft }) : undefined
          }
          onSelectProject={handleSelectProject}
          onSelectSchedule={handleSelectSchedule}
          onCreateProject={handleCreateProject}
          onCreateSchedule={handleCreateSchedule}
          onAccept={handleAccept}
        />
      ) : null}
    </div>
  );
}

const ADDED_AUTO_DISMISS_MS = 6000;

function AddedConfirmation({ scheduleName, projectUrl, onDismiss }: {
  scheduleName: string;
  projectUrl: string;
  onDismiss: () => void;
}) {
  useEffect(() => {
    const id = window.setTimeout(onDismiss, ADDED_AUTO_DISMISS_MS);
    return () => window.clearTimeout(id);
  }, [onDismiss]);

  return (
    <aside className="sally-panel added-confirmation" aria-label="Sally">
      <button className="added-dismiss" type="button" onClick={onDismiss} aria-label="Dismiss">×</button>
      <div className="added-body">
        <div className="added-check">✓</div>
        <p className="added-message">Added to <strong>{scheduleName}</strong></p>
        <a className="added-link action-button primary" href={projectUrl} target="_blank" rel="noopener noreferrer">
          View project →
        </a>
      </div>
      <div className="added-progress">
        <div className="added-progress-bar" style={{ animationDuration: `${ADDED_AUTO_DISMISS_MS}ms` }} />
      </div>
    </aside>
  );
}

function codePrefix(scheduleName: string): string {
  for (const char of scheduleName.toUpperCase()) {
    if (char >= "A" && char <= "Z") return char;
  }
  return "X";
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
