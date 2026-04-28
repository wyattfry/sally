import { useEffect, useRef, useState } from "react";
import { EXTRACT_TIMEOUT_MS } from "../lib/extractApi";
import type { ScheduleItem } from "../lib/types";

const TOTAL_TIMEOUT_SECONDS = Math.ceil(EXTRACT_TIMEOUT_MS / 1000);

type PanelState =
  | { kind: "thinking"; tokenCount: number }
  | { kind: "review"; draft: ScheduleItem };

type SallyPanelProps = {
  panel: PanelState;
  projectName: string;
  zones: string[];
  onChange: (draft: ScheduleItem) => void;
  onAddZone: (zone: string) => void;
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

const ADD_NEW_ZONE_VALUE = "__add_new__";
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
  projectName,
  zones,
  onChange,
  onAddZone,
  onAccept,
  onCancel,
  onViewItems
}: SallyPanelProps) {
  const draft = panel.kind === "review" ? panel.draft : undefined;
  const [isAddingZone, setIsAddingZone] = useState(false);
  const [newZone, setNewZone] = useState("");
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
        <div className="panel-kicker">{projectName}</div>
        <div className="panel-title">{panel.kind === "thinking" ? "Reading page" : "Add item to schedule"}</div>
        {draft ? <div className="panel-source">{draft.sourceTitle}</div> : null}
      </div>

      <div className="panel-body">
        {panel.kind === "thinking" ? (
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
        <button className="action-button secondary" type="button" onClick={onViewItems}>
          View Items
        </button>
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
