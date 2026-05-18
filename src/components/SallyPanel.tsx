import { useEffect, useRef, useState } from "react";
import type { ActiveContext, Project, Schedule, ScheduleColumn, ScheduleItem } from "../lib/types";
import {
  addMothershipScheduleColumn,
  renameMothershipScheduleColumn,
  deleteMothershipScheduleColumn,
  reorderMothershipScheduleColumns,
} from "../lib/mothershipApi";

type PanelState =
  | { kind: "thinking"; tokenCount: number }
  | { kind: "review"; draft: ScheduleItem }
  | { kind: "error"; message: string };

type SallyPanelProps = {
  panel: PanelState;
  projects: Project[];
  schedules: Schedule[];
  columns: ScheduleColumn[];
  rooms: string[];
  activeContext: ActiveContext | null;
  suggestedNewScheduleName?: string;
  onChange: (draft: ScheduleItem) => void;
  onSelectProject: (projectId: string) => void;
  onSelectSchedule: (scheduleId: string) => void;
  onCreateProject: (name: string) => Promise<string | null>;
  onCreateSchedule: (name: string) => Promise<string | null>;
  onColumnsChange: (cols: ScheduleColumn[]) => void;
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
  rooms,
  activeContext,
  suggestedNewScheduleName,
  onChange,
  onSelectProject,
  onSelectSchedule,
  onCreateProject,
  onCreateSchedule,
  onColumnsChange,
  onAccept,
  onCancel,
  onCancelAutoSchedule,
}: SallyPanelProps) {
  const draft = panel.kind === "review" ? panel.draft : undefined;
  const [modal, setModal] = useState<null | "project" | "schedule" | "room" | "image" | "columns">(null);
  const [modalInputValue, setModalInputValue] = useState("");
  const [modalError, setModalError] = useState<string | null>(null);
  const [modalAutoTriggered, setModalAutoTriggered] = useState(false);
  const modalInputRef = useRef<HTMLInputElement>(null);
  const [localRooms, setLocalRooms] = useState<string[]>(rooms);
  const [editColumns, setEditColumns] = useState<ScheduleColumn[]>([]);
  const [addColLabel, setAddColLabel] = useState("");
  const [colError, setColError] = useState<string | null>(null);

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
    if (modal === "columns") {
      onColumnsChange(editColumns);
    }
    setModal(null);
    setModalInputValue("");
    setModalError(null);
    setModalAutoTriggered(false);
  }

  async function handleMoveColumn(idx: number, dir: -1 | 1) {
    if (!activeContext?.projectId || !activeContext?.scheduleId) return;
    const next = [...editColumns];
    const [col] = next.splice(idx, 1);
    next.splice(idx + dir, 0, col);
    setEditColumns(next);
    try {
      await reorderMothershipScheduleColumns(activeContext.projectId, activeContext.scheduleId, next.map(c => c.id));
    } catch (e) {
      setColError(e instanceof Error ? e.message : "Reorder failed.");
    }
  }

  async function handleRenameColumn(colId: string, label: string) {
    if (!activeContext?.projectId || !activeContext?.scheduleId || !label.trim()) return;
    try {
      await renameMothershipScheduleColumn(activeContext.projectId, activeContext.scheduleId, colId, label.trim());
      setEditColumns(prev => prev.map(c => c.id === colId ? { ...c, label: label.trim() } : c));
    } catch (e) {
      setColError(e instanceof Error ? e.message : "Rename failed.");
    }
  }

  async function handleDeleteColumn(colId: string) {
    if (!activeContext?.projectId || !activeContext?.scheduleId) return;
    try {
      await deleteMothershipScheduleColumn(activeContext.projectId, activeContext.scheduleId, colId);
      setEditColumns(prev => prev.filter(c => c.id !== colId));
    } catch (e) {
      setColError(e instanceof Error ? e.message : "Delete failed.");
    }
  }

  async function handleAddColumn() {
    if (!addColLabel.trim() || !activeContext?.projectId || !activeContext?.scheduleId) return;
    try {
      const updated = await addMothershipScheduleColumn(activeContext.projectId, activeContext.scheduleId, addColLabel.trim());
      setEditColumns(updated.filter(c => c.key !== "room"));
      setAddColLabel("");
      setColError(null);
    } catch (e) {
      setColError(e instanceof Error ? e.message : "Could not add column.");
    }
  }

  async function submitModal() {
    const name = modalInputValue.trim();
    if (!name) return;
    if (modal === "room") {
      setLocalRooms((prev) => prev.includes(name) ? prev : [...prev, name]);
      onChange({ ...draft!, room: name });
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
            {modal === "columns" ? (
              <>
                <p className="panel-modal-title">Edit Columns</p>
                <ul className="col-edit-list">
                  {editColumns.map((col, i) => (
                    <li key={col.id} className="col-edit-row">
                      <div className="col-edit-move">
                        <button type="button" disabled={i === 0} onClick={() => handleMoveColumn(i, -1)}>↑</button>
                        <button type="button" disabled={i === editColumns.length - 1} onClick={() => handleMoveColumn(i, 1)}>↓</button>
                      </div>
                      <input
                        className="col-edit-label"
                        defaultValue={col.label}
                        onBlur={(e) => handleRenameColumn(col.id, e.target.value)}
                        onKeyDown={(e) => { if (e.key === "Enter") (e.target as HTMLInputElement).blur(); }}
                      />
                      <button
                        type="button"
                        className="col-edit-delete"
                        onClick={() => handleDeleteColumn(col.id)}
                      >Delete</button>
                    </li>
                  ))}
                </ul>
                <div className="inline-add">
                  <input
                    placeholder="New column name"
                    value={addColLabel}
                    onChange={(e) => { setAddColLabel(e.target.value); setColError(null); }}
                    onKeyDown={(e) => { if (e.key === "Enter") handleAddColumn(); }}
                  />
                  <button
                    className="action-button primary"
                    type="button"
                    disabled={!addColLabel.trim()}
                    onClick={handleAddColumn}
                  >Add</button>
                </div>
                {colError ? <p className="panel-modal-error">{colError}</p> : null}
                <div className="panel-modal-actions">
                  <button className="action-button primary" type="button" onClick={() => closeModal()}>Done</button>
                </div>
              </>
            ) : modal === "image" ? (
              <>
                <p className="panel-modal-title">Choose image</p>
                <div className="image-picker-grid">
                  {(draft?.sourceImageUrls ?? []).map((url) => (
                    <button
                      key={url}
                      type="button"
                      className={`image-picker-thumb${url === draft?.sourceImageUrl ? " selected" : ""}`}
                      onClick={() => {
                        onChange({ ...draft!, sourceImageUrl: url });
                        setModal(null);
                      }}
                    >
                      <img src={url} alt="" />
                    </button>
                  ))}
                </div>
                <div className="panel-modal-actions">
                  <button className="action-button secondary" type="button" onClick={() => setModal(null)}>
                    Cancel
                  </button>
                </div>
              </>
            ) : (
              <>
                <p className="panel-modal-title">
                  {modal === "project" ? "New project" : modal === "schedule" ? "New schedule" : "New room"}
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
              </>
            )}
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
              <button
                type="button"
                className="image-preview-btn"
                title={(draft.sourceImageUrls?.length ?? 0) > 1 ? "Click to choose image" : undefined}
                onClick={() => { if ((draft.sourceImageUrls?.length ?? 0) > 1) setModal("image"); }}
                style={(draft.sourceImageUrls?.length ?? 0) > 1 ? undefined : { cursor: "default" }}
              >
                <img className="image-preview" src={draft.sourceImageUrl} alt="" />
                {(draft.sourceImageUrls?.length ?? 0) > 1 && (
                  <span className="image-preview-count">{draft.sourceImageUrls!.length} photos</span>
                )}
              </button>
            ) : null}

            <div className="field">
              <label htmlFor="sally-room">Room</label>
              <select
                id="sally-room"
                value={draft?.room ?? ""}
                onChange={(event) => {
                  if (event.target.value === ADD_NEW_VALUE) {
                    setModal("room");
                    setModalInputValue("");
                    setModalError(null);
                    return;
                  }
                  onChange({ ...draft!, room: event.target.value });
                }}
              >
                <option value="">No room</option>
                {[...new Set([...(draft?.room ? [draft.room] : []), ...localRooms])].map((z) => (
                  <option key={z} value={z}>{z}</option>
                ))}
                <option value={ADD_NEW_VALUE}>New room...</option>
              </select>
            </div>

            {columns.filter((col) => col.key !== "room").map((col) => (
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
        {panel.kind === "review" && activeContext?.scheduleId ? (
          <button
            className="action-button secondary"
            type="button"
            style={{ marginRight: "auto" }}
            onClick={() => {
              setEditColumns(columns.filter(c => c.key !== "room"));
              setAddColLabel("");
              setColError(null);
              setModal("columns");
            }}
          >
            Edit Columns
          </button>
        ) : null}
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
