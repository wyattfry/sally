import { useState } from "react";
import type { ScheduleItem } from "../lib/types";

type PanelState =
  | { kind: "thinking" }
  | { kind: "review"; draft: ScheduleItem };

type SallyPanelProps = {
  panel: PanelState;
  zones: string[];
  onChange: (draft: ScheduleItem) => void;
  onAddZone: (zone: string) => void;
  onAccept: (draft: ScheduleItem) => void;
  onCancel: () => void;
};

const textFields = [
  ["title", "Title"],
  ["manufacturer", "Manufacturer"],
  ["modelNumber", "Model"],
  ["category", "Category"],
  ["finish", "Finish"],
  ["finishModelNumber", "Finish Model"]
] as const;

const ADD_NEW_ZONE_VALUE = "__add_new__";

export function SallyPanel({ panel, zones, onChange, onAddZone, onAccept, onCancel }: SallyPanelProps) {
  const draft = panel.kind === "review" ? panel.draft : undefined;
  const [isAddingZone, setIsAddingZone] = useState(false);
  const [newZone, setNewZone] = useState("");

  function updateField<Key extends keyof ScheduleItem>(key: Key, value: ScheduleItem[Key]) {
    if (!draft) {
      return;
    }
    onChange({ ...draft, [key]: value });
  }

  return (
    <aside className="sally-panel" aria-label="Sally proposal">
      <div className="panel-header">
        <div className="panel-kicker">Sally proposal</div>
        <div className="panel-title">{panel.kind === "thinking" ? "Reading page" : "Review item"}</div>
        {draft ? <div className="panel-source">{draft.sourceTitle}</div> : null}
      </div>

      <div className="panel-body">
        {panel.kind === "thinking" ? (
          <div className="thinking">Reading product information and drafting a schedule item.</div>
        ) : (
          <>
            {draft?.sourceImageUrl ? (
              <img className="image-preview" src={draft.sourceImageUrl} alt="" />
            ) : null}

            <div className="field">
              <label htmlFor="sally-zone">Zone</label>
              <select
                id="sally-zone"
                value={isAddingZone ? ADD_NEW_ZONE_VALUE : draft?.zone ?? ""}
                onChange={(event) => {
                  if (event.target.value === ADD_NEW_ZONE_VALUE) {
                    setIsAddingZone(true);
                    return;
                  }

                  setIsAddingZone(false);
                  updateField("zone", event.target.value);
                }}
              >
                <option value="">Unassigned</option>
                {zones.map((zone) => (
                  <option key={zone} value={zone}>
                    {zone}
                  </option>
                ))}
                {draft?.zone && !zones.includes(draft.zone) ? (
                  <option value={draft.zone}>{draft.zone}</option>
                ) : null}
                <option value={ADD_NEW_ZONE_VALUE}>Add new zone...</option>
              </select>
            </div>

            {isAddingZone ? (
              <div className="inline-add">
                <div className="field">
                  <label htmlFor="sally-new-zone">New zone</label>
                  <input
                    id="sally-new-zone"
                    value={newZone}
                    onChange={(event) => setNewZone(event.target.value)}
                  />
                </div>
                <button
                  className="action-button secondary"
                  disabled={!newZone.trim()}
                  type="button"
                  onClick={() => {
                    onAddZone(newZone);
                    setNewZone("");
                    setIsAddingZone(false);
                  }}
                >
                  Add zone
                </button>
              </div>
            ) : null}

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
                value={draft?.requiredAddOns.join(", ") ?? ""}
                onChange={(event) =>
                  updateField("requiredAddOns", splitList(event.target.value))
                }
              />
            </div>

            <div className="field">
              <label htmlFor="sally-optional-companions">Optional Companions</label>
              <input
                id="sally-optional-companions"
                value={draft?.optionalCompanions.join(", ") ?? ""}
                onChange={(event) =>
                  updateField("optionalCompanions", splitList(event.target.value))
                }
              />
            </div>

            {draft?.sourcePdfLinks.length ? (
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
          disabled={!draft}
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
