import { useEffect, useRef, useState } from "react";
import { EXTRACT_TIMEOUT_MS } from "../lib/extractApi";
import type { ActiveContext, Project, Schedule, ScheduleItem } from "../lib/types";

const TOTAL_TIMEOUT_SECONDS = Math.ceil(EXTRACT_TIMEOUT_MS / 1000);

type PanelState =
  | { kind: "thinking"; tokenCount: number }
  | { kind: "review"; draft: ScheduleItem }
  | { kind: "error"; message: string };

type SallyPanelProps = {
  panel: PanelState;
  projects: Project[];
  schedules: Schedule[];
  activeContext: ActiveContext | null;
  onChange: (draft: ScheduleItem) => void;
  onSelectProject: (projectId: string) => void;
  onSelectSchedule: (scheduleId: string) => void;
  onCreateProject: (name: string) => Promise<string | null>;
  onCreateSchedule: (name: string) => Promise<string | null>;
  onAccept: (draft: ScheduleItem) => void;
  onCancel: () => void;
  onViewItems: () => void;
};

const textFields = [
  ["title", "Title"],
  ["manufacturer", "Manufacturer"],
  ["modelNumber", "Model"],
  ["finish", "Finish"],
  ["finishModelNumber", "Finish Model"]
] as const;

const ADD_NEW_VALUE = "__add_new__";
const DEFAULT_CATEGORIES = [
  "Plumbing Fixture",
  "Lighting",
  "Appliance",
  "Hardware",
  "Finish",
  "Furniture",
  "Accessory"
];

export function SallyPanel({
  panel,
  projects,
  schedules,
  activeContext,
  onChange,
  onSelectProject,
  onSelectSchedule,
  onCreateProject,
  onCreateSchedule,
  onAccept,
  onCancel,
  onViewItems
}: SallyPanelProps) {
  const draft = panel.kind === "review" ? panel.draft : undefined;
  const [isAddingProject, setIsAddingProject] = useState(false);
  const [newProjectName, setNewProjectName] = useState("");
  const [projectCreateError, setProjectCreateError] = useState<string | null>(null);
  const [isAddingSchedule, setIsAddingSchedule] = useState(false);
  const [newScheduleName, setNewScheduleName] = useState("");
  const [scheduleCreateError, setScheduleCreateError] = useState<string | null>(null);
  const [secondsLeft, setSecondsLeft] = useState(TOTAL_TIMEOUT_SECONDS);
  const intervalRef = useRef<number | null>(null);

  useEffect(() => {
    if (panel.kind !== "thinking") {
      setSecondsLeft(TOTAL_TIMEOUT_SECONDS);
      return;
    }
    setSecondsLeft(TOTAL_TIMEOUT_SECONDS);
    intervalRef.current = window.setInterval(() => {
      setSecondsLeft((s) => Math.max(0, s - 1));
    }, 1000);
    return () => {
      if (intervalRef.current !== null) window.clearInterval(intervalRef.current);
    };
  }, [panel.kind]);

  function updateField<Key extends keyof ScheduleItem>(key: Key, value: ScheduleItem[Key]) {
    if (!draft) {
      return;
    }
    onChange({ ...draft, [key]: value });
  }

  return (
    <aside className="sally-panel" aria-label="Sally capture panel">
      <div className="panel-header">
        <div className="panel-context">
          <div className="field">
            <label htmlFor="sally-project">Project</label>
            <select
              id="sally-project"
              value={isAddingProject ? ADD_NEW_VALUE : activeContext?.projectId ?? ""}
              onChange={(event) => {
                if (event.target.value === ADD_NEW_VALUE) {
                  setIsAddingProject(true);
                  return;
                }
                setIsAddingProject(false);
                onSelectProject(event.target.value);
              }}
            >
              <option value="" disabled>Select a project...</option>
              {projects.map((project) => (
                <option key={project.id} value={project.id}>
                  {project.name}
                </option>
              ))}
              <option value={ADD_NEW_VALUE}>New project...</option>
            </select>
          </div>
          {isAddingProject ? (
            <div className="inline-add">
              <input
                id="sally-new-project"
                placeholder="Project name"
                value={newProjectName}
                onChange={(event) => {
                  setNewProjectName(event.target.value);
                  setProjectCreateError(null);
                }}
              />
              <button
                className="action-button secondary"
                disabled={!newProjectName.trim()}
                type="button"
                onClick={async () => {
                  const error = await onCreateProject(newProjectName.trim());
                  if (error) {
                    setProjectCreateError(error);
                  } else {
                    setNewProjectName("");
                    setIsAddingProject(false);
                    setProjectCreateError(null);
                  }
                }}
              >
                Add
              </button>
              {projectCreateError ? (
                <span className="inline-add-error">{projectCreateError}</span>
              ) : null}
            </div>
          ) : null}

          <div className="field">
            <label htmlFor="sally-schedule">Schedule</label>
            <select
              id="sally-schedule"
              value={isAddingSchedule ? ADD_NEW_VALUE : activeContext?.scheduleId ?? ""}
              onChange={(event) => {
                if (event.target.value === ADD_NEW_VALUE) {
                  setIsAddingSchedule(true);
                  return;
                }
                setIsAddingSchedule(false);
                onSelectSchedule(event.target.value);
              }}
            >
              <option value="" disabled>Select a schedule...</option>
              {schedules.map((schedule) => (
                <option key={schedule.id} value={schedule.id}>
                  {schedule.name}
                </option>
              ))}
              {activeContext?.projectId ? <option value={ADD_NEW_VALUE}>New schedule...</option> : null}
            </select>
          </div>
          {isAddingSchedule ? (
            <div className="inline-add">
              <input
                id="sally-new-schedule"
                placeholder="Schedule name"
                value={newScheduleName}
                onChange={(event) => {
                  setNewScheduleName(event.target.value);
                  setScheduleCreateError(null);
                }}
              />
              <button
                className="action-button secondary"
                disabled={!newScheduleName.trim()}
                type="button"
                onClick={async () => {
                  const error = await onCreateSchedule(newScheduleName.trim());
                  if (error) {
                    setScheduleCreateError(error);
                  } else {
                    setNewScheduleName("");
                    setIsAddingSchedule(false);
                    setScheduleCreateError(null);
                  }
                }}
              >
                Add
              </button>
              {scheduleCreateError ? (
                <span className="inline-add-error">{scheduleCreateError}</span>
              ) : null}
            </div>
          ) : null}
        </div>
        <div className="panel-title">{panel.kind === "thinking" ? "Reading page" : panel.kind === "error" ? "Error" : "Add item to schedule"}</div>
        {draft ? <div className="panel-source">{draft.sourceTitle}</div> : null}
      </div>

      <div className="panel-body">
        {panel.kind === "error" ? (
          <div className="sally-error-state">
            <p>{panel.message}</p>
          </div>
        ) : panel.kind === "thinking" ? (
          <div className="thinking">
            <div aria-hidden="true" className="thinking-spinner" />
            <p>
              {panel.tokenCount > 0
                ? `Generating response… (${panel.tokenCount} tokens)`
                : "Reading product information and drafting a schedule item."}
            </p>
            <div className={`thinking-countdown${secondsLeft <= 10 ? " thinking-countdown--urgent" : ""}`}>
              {secondsLeft}s remaining
            </div>
          </div>
        ) : (
          <>
            {draft?.sourceImageUrl ? (
              <img className="image-preview" src={draft.sourceImageUrl} alt="" />
            ) : null}

            <div className="field">
              <label htmlFor="sally-category">Category</label>
              <select
                id="sally-category"
                value={draft?.category ?? ""}
                onChange={(event) => updateField("category", event.target.value)}
              >
                {draft?.category && !DEFAULT_CATEGORIES.includes(draft.category) ? (
                  <option value={draft.category}>{draft.category}</option>
                ) : null}
                {DEFAULT_CATEGORIES.map((category) => (
                  <option key={category} value={category}>
                    {category}
                  </option>
                ))}
              </select>
            </div>

            {textFields.map(([key, label]) => (
              <div className="field" key={key}>
                <label htmlFor={`sally-${key}`}>{label}</label>
                <input
                  id={`sally-${key}`}
                  value={String(draft?.[key] ?? "")}
                  onChange={(event) => updateField(key, event.target.value)}
                />
              </div>
            ))}

            <div className="field">
              <label htmlFor="sally-description">Description</label>
              <textarea
                id="sally-description"
                value={draft?.description ?? ""}
                onChange={(event) => updateField("description", event.target.value)}
              />
            </div>

            <div className="field">
              <label htmlFor="sally-required-addons">Required Add-ons</label>
              <input
                id="sally-required-addons"
                value={(draft?.requiredAddOns ?? []).join(", ")}
                onChange={(event) =>
                  updateField("requiredAddOns", splitList(event.target.value))
                }
              />
            </div>

            <div className="field">
              <label htmlFor="sally-optional-companions">Optional Companions</label>
              <input
                id="sally-optional-companions"
                value={(draft?.optionalCompanions ?? []).join(", ")}
                onChange={(event) =>
                  updateField("optionalCompanions", splitList(event.target.value))
                }
              />
            </div>

            {draft?.sourcePdfLinks?.length ? (
              <div className="source-links">
                {draft.sourcePdfLinks.map((link) => (
                  <a className="source-link" href={link} key={link} rel="noreferrer" target="_blank">
                    {link}
                  </a>
                ))}
              </div>
            ) : null}
          </>
        )}
      </div>

      <div className="panel-actions">
        <button className="action-button secondary" type="button" onClick={onViewItems}>
          View Items
        </button>
        <button className="action-button secondary" type="button" onClick={onCancel}>
          Cancel
        </button>
        <button
          className="action-button primary"
          disabled={!draft || !activeContext?.scheduleId}
          type="button"
          onClick={() => draft && onAccept(draft)}
        >
          OK
        </button>
      </div>
    </aside>
  );
}

function splitList(value: string): string[] {
  return value
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}
