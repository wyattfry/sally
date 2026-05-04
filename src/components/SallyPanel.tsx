import { useEffect, useRef, useState } from "react";
import type { ActiveContext, Project, Schedule, ScheduleItem } from "../lib/types";

type PanelState =
  | { kind: "thinking"; tokenCount: number }
  | { kind: "review"; draft: ScheduleItem }
  | { kind: "error"; message: string };

type SallyPanelProps = {
  panel: PanelState;
  projects: Project[];
  schedules: Schedule[];
  zones: string[];
  activeContext: ActiveContext | null;
  suggestedNewScheduleName?: string;
  onChange: (draft: ScheduleItem) => void;
  onSelectProject: (projectId: string) => void;
  onSelectSchedule: (scheduleId: string) => void;
  onCreateProject: (name: string) => Promise<string | null>;
  onCreateSchedule: (name: string) => Promise<string | null>;
  onAccept: (draft: ScheduleItem) => void;
  onCancel: () => void;
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
  zones,
  activeContext,
  suggestedNewScheduleName,
  onChange,
  onSelectProject,
  onSelectSchedule,
  onCreateProject,
  onCreateSchedule,
  onAccept,
  onCancel,
}: SallyPanelProps) {
  const draft = panel.kind === "review" ? panel.draft : undefined;
  const [modal, setModal] = useState<null | "project" | "schedule" | "zone">(null);
  const [modalInputValue, setModalInputValue] = useState("");
  const [modalError, setModalError] = useState<string | null>(null);
  const modalInputRef = useRef<HTMLInputElement>(null);
  const [localZones, setLocalZones] = useState<string[]>(zones);

  useEffect(() => {
    if (suggestedNewScheduleName) {
      setModal("schedule");
      setModalInputValue(suggestedNewScheduleName);
      setModalError(null);
    }
  }, [suggestedNewScheduleName]);

  useEffect(() => {
    if (modal) {
      modalInputRef.current?.focus();
      modalInputRef.current?.select();
    }
  }, [modal]);


  function updateField<Key extends keyof ScheduleItem>(key: Key, value: ScheduleItem[Key]) {
    if (!draft) return;
    onChange({ ...draft, [key]: value });
  }

  function closeModal() {
    setModal(null);
    setModalInputValue("");
    setModalError(null);
  }

  async function submitModal() {
    const name = modalInputValue.trim();
    if (!name) return;
    if (modal === "zone") {
      setLocalZones((prev) => prev.includes(name) ? prev : [...prev, name]);
      updateField("zone", name);
      closeModal();
      return;
    }
    const error = modal === "project"
      ? await onCreateProject(name)
      : await onCreateSchedule(name);
    if (error) {
      setModalError(error);
    } else {
      closeModal();
    }
  }

  return (
    <aside className="sally-panel" aria-label="Sally capture panel">
      {modal ? (
        <div
          className="panel-modal-backdrop"
          onClick={(e) => { if (e.target === e.currentTarget) closeModal(); }}
        >
          <div className="panel-modal" role="dialog" aria-modal="true">
            <p className="panel-modal-title">
              {modal === "project" ? "New project" : modal === "schedule" ? "New schedule" : "New zone"}
            </p>
            <div className="field">
              <label htmlFor="panel-modal-input">Name</label>
              <input
                id="panel-modal-input"
                ref={modalInputRef}
                value={modalInputValue}
                placeholder={modal === "project" ? "Project name" : "Schedule name"}
                onChange={(e) => { setModalInputValue(e.target.value); setModalError(null); }}
                onKeyDown={(e) => {
                  if (e.key === "Enter") submitModal();
                  if (e.key === "Escape") closeModal();
                }}
              />
            </div>
            {modalError ? <p className="panel-modal-error">{modalError}</p> : null}
            <div className="panel-modal-actions">
              <button className="action-button secondary" type="button" onClick={closeModal}>
                Cancel
              </button>
              <button
                className="action-button primary"
                type="button"
                disabled={!modalInputValue.trim()}
                onClick={submitModal}
              >
                Create
              </button>
            </div>
          </div>
        </div>
      ) : null}

      <div className="panel-header">
        <div className="panel-context">
          <div className="field">
            <label htmlFor="sally-project">Project</label>
            <select
              id="sally-project"
              value={activeContext?.projectId ?? ""}
              onChange={(event) => {
                if (event.target.value === ADD_NEW_VALUE) {
                  setModal("project");
                  setModalInputValue("");
                  setModalError(null);
                  return;
                }
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

          {panel.kind !== "thinking" ? (
            <div className="field">
              <label htmlFor="sally-schedule">Schedule</label>
              <select
                id="sally-schedule"
                value={activeContext?.scheduleId ?? ""}
                onChange={(event) => {
                  if (event.target.value === ADD_NEW_VALUE) {
                    setModal("schedule");
                    setModalInputValue("");
                    setModalError(null);
                    return;
                  }
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
          </div>
        ) : (
          <>
            {draft?.sourceImageUrl ? (
              <img className="image-preview" src={draft.sourceImageUrl} alt="" />
            ) : null}

            <div className="field">
              <label htmlFor="sally-zone">Zone</label>
              <select
                id="sally-zone"
                value={draft?.zone ?? ""}
                onChange={(event) => {
                  if (event.target.value === ADD_NEW_VALUE) {
                    setModal("zone");
                    setModalInputValue("");
                    setModalError(null);
                    return;
                  }
                  updateField("zone", event.target.value);
                }}
              >
                <option value="">No zone</option>
                {[...new Set([...(draft?.zone ? [draft.zone] : []), ...localZones])].map((z) => (
                  <option key={z} value={z}>{z}</option>
                ))}
                <option value={ADD_NEW_VALUE}>New zone...</option>
              </select>
            </div>

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
