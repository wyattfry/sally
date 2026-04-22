import type { ScheduleItem } from "../lib/types";

type ScheduleViewerProps = {
  items: ScheduleItem[];
  onClose: () => void;
};

export function ScheduleViewer({ items, onClose }: ScheduleViewerProps) {
  return (
    <aside className="schedule-viewer" aria-label="Captured schedule">
      <div className="viewer-header">
        <div>
          <div className="panel-kicker">Sally PoC</div>
          <div className="panel-title">Captured schedule</div>
        </div>
        <button className="icon-button" type="button" onClick={onClose} aria-label="Close schedule">
          x
        </button>
      </div>

      <div className="viewer-list">
        {items.length === 0 ? (
          <div className="empty-state">No accepted items yet.</div>
        ) : (
          items.map((item) => (
            <article className="schedule-item" key={item.id}>
              <div className="schedule-thumb">
                {item.sourceImageUrl ? (
                  <img src={item.sourceImageUrl} alt={`${item.title} thumbnail`} />
                ) : (
                  <div aria-label={`${item.title} thumbnail unavailable`} role="img">
                    No image
                  </div>
                )}
              </div>
              <div className="schedule-item-content">
                <div className="schedule-item-topline">
                  <span>{item.zone || "Unassigned"}</span>
                  <span>{item.category}</span>
                </div>
                <h2>{item.title}</h2>
                <div className="schedule-item-grid">
                  <span>Manufacturer</span>
                  <strong>{item.manufacturer || "-"}</strong>
                  <span>Model</span>
                  <strong>{item.modelNumber || "-"}</strong>
                  <span>Finish</span>
                  <strong>{item.finish || "-"}</strong>
                </div>
                <p>{item.description}</p>
                <a href={item.sourceUrl} target="_blank" rel="noreferrer">
                  {new URL(item.sourceUrl).hostname}
                </a>
              </div>
            </article>
          ))
        )}
      </div>
    </aside>
  );
}
