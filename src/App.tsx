import { useEffect, useState } from "react";
import { SallyPanel } from "./components/SallyPanel";
import { ScheduleViewer } from "./components/ScheduleViewer";
import { SpecButton } from "./components/SpecButton";
import { capturePage } from "./lib/capturePage";
import { extractScheduleItem } from "./lib/extractApi";
import {
  getProjectName,
  listScheduleItems,
  listZones,
  removeScheduleItem,
  saveProjectName,
  saveScheduleItem,
  saveZone
} from "./lib/storage";
import type { ScheduleItem } from "./lib/types";

type PanelState =
  | { kind: "closed" }
  | { kind: "thinking" }
  | { kind: "review"; draft: ScheduleItem }
  | { kind: "minimized"; draft: ScheduleItem };

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
  const [toast, setToast] = useState<string | null>(null);

  useEffect(() => {
    refreshItemCount();
    refreshProjectName();
    refreshZones();
  }, []);

  useEffect(() => {
    if (!globalThis.chrome?.runtime?.onMessage) {
      return;
    }

    function handleRuntimeMessage(message: unknown) {
      if (isSpecMessage(message)) {
        handleSpecClick();
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

  function handleSpecClick() {
    setPanel({ kind: "thinking" });
    window.setTimeout(async () => {
      const captured = capturePage(document, window.location);
      try {
        const proposal = await extractScheduleItem({
          capturedPage: captured,
          projectName,
          knownZones: zones,
          knownCategories: DEFAULT_CATEGORIES
        });
        setPanel({ kind: "review", draft: proposal });
      } catch (error) {
        setPanel({ kind: "closed" });
        showToast(error instanceof Error ? error.message : "Could not extract item.");
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
    await saveScheduleItem(item);
    await refreshItemCount();
    setPanel({ kind: "closed" });
    showToast("Item added");
  }

  async function handleRenameProject(nextProjectName: string) {
    setProjectName(await saveProjectName(nextProjectName));
  }

  async function handleRemoveItem(itemId: string) {
    const items = await removeScheduleItem(itemId);
    setScheduleItems(items);
  }

  function handleViewItems() {
    setPanel({ kind: "closed" });
    setIsScheduleOpen(true);
  }

  function showToast(message: string) {
    setToast(message);
    window.setTimeout(() => setToast(null), 3200);
  }

  return (
    <div className="sally-root">
      <SpecButton
        onClick={handleSpecClick}
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
      {panel.kind === "thinking" || panel.kind === "review" ? (
        <SallyPanel
          panel={panel}
          projectName={projectName}
          zones={zones}
          onCancel={() => setPanel({ kind: "closed" })}
          onChange={(draft) =>
            panel.kind === "review" ? setPanel({ ...panel, draft }) : undefined
          }
          onAddZone={handleAddZone}
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
