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
        const { item, suggestedScheduleName } = await extractScheduleItem({
          capturedPage: captured,
          knownCategories: [],
          knownScheduleNames,
          scheduleId: activeContext?.scheduleId,
          onProgress: (tokenCount) => {
            setPanel((prev) => prev.kind === "thinking" ? { kind: "thinking", tokenCount } : prev);
          }
        });
        const matchingSchedule = suggestedScheduleName
          ? schedules.find((s) => s.name.toLowerCase() === suggestedScheduleName.toLowerCase())
          : undefined;
        if (matchingSchedule && matchingSchedule.id !== activeContext?.scheduleId) {
          await handleSelectSchedule(matchingSchedule.id);
        }
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
      setPanel({ kind: "closed" });
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
      const fetchedColumns = await listMothershipScheduleColumns(scheduleId);
      setColumns(fetchedColumns);
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
      ) : panel.kind !== "closed" && panel.kind !== "minimized" ? (
        <SallyPanel
          panel={panel}
          projects={projects}
          schedules={schedules}
          columns={columns}
          zones={[]}
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
