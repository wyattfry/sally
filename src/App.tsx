import { useEffect, useState } from "react";
import { SallyPanel } from "./components/SallyPanel";
import { SpecButton } from "./components/SpecButton";
import { capturePage } from "./lib/capturePage";
import { mockExtractScheduleItem } from "./lib/mockExtraction";
import { listScheduleItems, saveScheduleItem } from "./lib/storage";
import type { ScheduleItem } from "./lib/types";

type PanelState =
  | { kind: "closed" }
  | { kind: "thinking" }
  | { kind: "review"; original: ScheduleItem; draft: ScheduleItem };

export default function App() {
  const [panel, setPanel] = useState<PanelState>({ kind: "closed" });
  const [itemCount, setItemCount] = useState(0);

  useEffect(() => {
    refreshItemCount();
  }, []);

  async function refreshItemCount() {
    const items = await listScheduleItems();
    setItemCount(items.length);
  }

  function handleSpecClick() {
    setPanel({ kind: "thinking" });
    window.setTimeout(() => {
      const captured = capturePage(document, window.location);
      const proposal = mockExtractScheduleItem(captured);
      setPanel({ kind: "review", original: proposal, draft: proposal });
    }, 250);
  }

  async function handleAccept(item: ScheduleItem) {
    await saveScheduleItem(item);
    await refreshItemCount();
    setPanel({ kind: "closed" });
  }

  return (
    <div className="sally-root">
      <SpecButton itemCount={itemCount} onClick={handleSpecClick} />
      {panel.kind !== "closed" ? (
        <SallyPanel
          panel={panel}
          onCancel={() => setPanel({ kind: "closed" })}
          onChange={(draft) =>
            panel.kind === "review" ? setPanel({ ...panel, draft }) : undefined
          }
          onUndo={() =>
            panel.kind === "review" ? setPanel({ ...panel, draft: panel.original }) : undefined
          }
          onAccept={handleAccept}
        />
      ) : null}
    </div>
  );
}
