import { useEffect, useRef, useState } from "react";
import type { ActiveContext, Project, Schedule, ScheduleColumn, ScheduleItem } from "../lib/types";

type PanelState =
  | { kind: "thinking"; tokenCount: number }
  | { kind: "review"; draft: ScheduleItem }
  | { kind: "error"; message: string };

type SallyPanelProps = {
  panel: PanelState;
  projects: Project[];
  schedules: Schedule[];
  columns: ScheduleColumn[];
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
  onCancelAutoSchedule: () => void;
};

const ADD_NEW_VALUE = "__add_new__";

export function SallyPanel({
  panel,
  projects,
  schedules,
  columns,
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
  onCancelAutoSchedule,
}: SallyPanelProps) {
  const draft = panel.kind === "review" ? panel.draft : undefined;
  const [modal, setModal] = useState<null | "project" | "schedule" | "zone">(null);
  const [modalInputValue, setModalInputValue] = useState("");
  const [modalError, setModalError] = useState<string | null>(null);
  const [modalAutoTriggered, setModalAutoTriggered] = useState(false);
  const modalInputRef = useRef<HTMLInputElement>(null);
  const [localZones, setLocalZones] = useState<string[]>(zones);

  useEffect(() => {
    if (suggestedNewScheduleName) {
      setModal("schedule");
      setModalInputValue(suggestedNewScheduleName);
      setModalError(null);
      setModalAutoTriggered(true);
    }
  }, [suggestedNewScheduleName]);

  useEffect(() => {
    if (modal) {
      modalInputRef.current?.focus();
      modalInputRef.current?.select();
    }
  }, [modal]);

  function updateData(key: string, value: string) {
    if (!draft) return;
    onChange({ ...draft, data: { ...draft.data, [key]: value } });
  }

  function closeModal({ created = false } = {}) {
    if (!created && modalAutoTriggered && modal === "schedule") {
      onCancelAutoSchedule();
    }
    setModal(null);
    setModalInputValue("");
    setModalError(null);
    setModalAutoTriggered(false);
  }

  async function submitModal() {
    const name = modalInputValue.trim();
    if (!name) return;
    if (modal === "zone") {
      setLocalZones((prev) => prev.includes(name) ? prev : [...prev, name]);
      onChange({ ...draft!, zone: name });
      closeModal();
      return;
    }
    const error = modal === "project"
      ? await onCreateProject(name)
      : await onCreateSchedule(name);
    if (error) {
      setModalError(error);
    } else {
      closeModal({ created: true });
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
            {modal === "schedule" && modalAutoTriggered && (
              <p className="panel-modal-hint">
                This item doesn't seem to belong in any of your existing schedules. Create a new one?
              </p>
            )}
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
              <button className="action-button secondary" type="button" onClick={() => closeModal()}>
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
        <div className="panel-title">Add item to schedule</div>
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
                    setModalAutoTriggered(false);
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
                  onChange({ ...draft!, zone: event.target.value });
                }}
              >
                <option value="">No zone</option>
                {[...new Set([...(draft?.zone ? [draft.zone] : []), ...localZones])].map((z) => (
                  <option key={z} value={z}>{z}</option>
                ))}
                <option value={ADD_NEW_VALUE}>New zone...</option>
              </select>
            </div>

            {columns.filter((col) => col.key !== "zone").map((col) => (
              <div className="field" key={col.key}>
                <label htmlFor={`sally-col-${col.key}`}>{col.label}</label>
                {col.key === "code"
                  ? <CodeField
                      id={`sally-col-${col.key}`}
                      value={draft?.data[col.key] ?? ""}
                      onChange={(v) => updateData(col.key, v)}
                    />
                  : <input
                      id={`sally-col-${col.key}`}
                      value={draft?.data[col.key] ?? ""}
                      onChange={(event) => updateData(col.key, event.target.value)}
                    />
                }
              </div>
            ))}

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

function parseCode(value: string): { prefix: string; suffix: string } | null {
  const i = value.lastIndexOf("-");
  if (i <= 0) return null;
  const suffix = value.slice(i + 1);
  if (!/^\d+$/.test(suffix)) return null;
  return { prefix: value.slice(0, i), suffix };
}

function CodeField({ id, value, onChange }: { id: string; value: string; onChange: (v: string) => void }) {
  const parsed = parseCode(value);
  if (!parsed) {
    return <input id={id} value={value} onChange={(e) => onChange(e.target.value)} />;
  }
  const { prefix, suffix } = parsed;
  return (
    <div className="code-field">
      <span className="code-prefix" aria-hidden="true">{prefix}-</span>
      <input
        id={id}
        className="code-suffix"
        value={suffix}
        inputMode="numeric"
        onChange={(e) => {
          const n = e.target.value.replace(/\D/g, "");
          onChange(`${prefix}-${n || "1"}`);
        }}
      />
    </div>
  );
}
