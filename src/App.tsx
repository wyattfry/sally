import { useEffect, useState } from "react";
import { SallyPanel } from "./components/SallyPanel";
import { ScheduleViewer } from "./components/ScheduleViewer";
import { SpecButton } from "./components/SpecButton";
import { capturePage } from "./lib/capturePage";
import { mockExtractScheduleItem } from "./lib/mockExtraction";
import { listScheduleItems, listZones, saveScheduleItem, saveZone } from "./lib/storage";
import type { ScheduleItem } from "./lib/types";

type PanelState =
  | { kind: "closed" }
  | { kind: "thinking" }
  | { kind: "review"; draft: ScheduleItem }
  | { kind: "minimized"; draft: ScheduleItem };

export default function App() {
  const [panel, setPanel] = useState<PanelState>({ kind: "closed" });
  const [isScheduleOpen, setIsScheduleOpen] = useState(false);
  const [itemCount, setItemCount] = useState(0);
  const [isCurrentPageSpecd, setIsCurrentPageSpecd] = useState(false);
  const [scheduleItems, setScheduleItems] = useState<ScheduleItem[]>([]);
  const [zones, setZones] = useState<string[]>([]);

  useEffect(() => {
    refreshItemCount();
    refreshZones();
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
    setItemCount(items.length);
    setIsCurrentPageSpecd(items.some((item) => samePageUrl(item.sourceUrl, window.location.href)));
  }

  async function refreshZones() {
    setZones(await listZones());
  }

  function handleSpecClick() {
    setPanel({ kind: "thinking" });
    window.setTimeout(() => {
      const captured = capturePage(document, window.location);
      const proposal = mockExtractScheduleItem(captured);
      setPanel({ kind: "review", draft: proposal });
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
  }

  return (
    <div className="sally-root">
      <SpecButton
        isCurrentPageSpecd={isCurrentPageSpecd}
        itemCount={itemCount}
        onOpenSchedule={() => setIsScheduleOpen(true)}
        onClick={handleSpecClick}
      />
      {isScheduleOpen ? (
        <ScheduleViewer items={scheduleItems} onClose={() => setIsScheduleOpen(false)} />
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
          zones={zones}
          onCancel={() => setPanel({ kind: "closed" })}
          onChange={(draft) =>
            panel.kind === "review" ? setPanel({ ...panel, draft }) : undefined
          }
          onAddZone={handleAddZone}
          onAccept={handleAccept}
        />
      ) : null}
    </div>
  );
}

function samePageUrl(left: string, right: string): boolean {
  try {
    const leftUrl = new URL(left);
    const rightUrl = new URL(right);
    leftUrl.hash = "";
    rightUrl.hash = "";
    return leftUrl.href === rightUrl.href;
  } catch {
    return left === right;
  }
}
