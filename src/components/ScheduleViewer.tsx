import { useState } from "react";
import type { ScheduleItem } from "../lib/types";

type ScheduleViewerProps = {
  items: ScheduleItem[];
  projectName: string;
  onClose: () => void;
  onRemoveItem: (itemId: string) => void;
  onRenameProject: (projectName: string) => void;
};

export function ScheduleViewer({
  items,
  projectName,
  onClose,
  onRemoveItem,
  onRenameProject
}: ScheduleViewerProps) {
  const [isRenaming, setIsRenaming] = useState(false);
  const [draftProjectName, setDraftProjectName] = useState(projectName);

  function handlePrint() {
    printSchedule(items, projectName);
  }

  function commitProjectName() {
    onRenameProject(draftProjectName);
    setIsRenaming(false);
  }

  return (
    <aside className="schedule-viewer" aria-label="Captured schedule">
      <div className="viewer-header">
        <div>
          {isRenaming ? (
            <input
              autoFocus
              className="project-name-input"
              aria-label="Project name"
              value={draftProjectName}
              onBlur={commitProjectName}
              onChange={(event) => setDraftProjectName(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === "Enter") {
                  commitProjectName();
                }
                if (event.key === "Escape") {
                  setDraftProjectName(projectName);
                  setIsRenaming(false);
                }
              }}
            />
          ) : (
            <button
              className="project-name-button"
              type="button"
              aria-label={`Rename ${projectName}`}
              onClick={() => {
                setDraftProjectName(projectName);
                setIsRenaming(true);
              }}
            >
              <span>{projectName}</span>
            </button>
          )}
        </div>
        <div className="viewer-actions">
          <button className="action-button secondary" type="button" onClick={handlePrint}>
            Print
          </button>
          <button className="icon-button" type="button" onClick={onClose} aria-label="Close schedule">
            ×
          </button>
        </div>
      </div>

      <div className="viewer-list">
        {items.length === 0 ? (
          <div className="empty-state">No accepted items yet.</div>
        ) : (
          items.map((item) => (
            <article className="schedule-item" key={item.id}>
              <a
                className="schedule-thumb"
                href={item.sourceUrl}
                target="_blank"
                rel="noreferrer"
                aria-label={`${item.title} thumbnail`}
              >
                {item.sourceImageUrl ? (
                  <img src={item.sourceImageUrl} alt="" />
                ) : (
                  <div>No image</div>
                )}
              </a>
              <div className="schedule-item-content">
                <div className="schedule-item-heading">
                  <div className="schedule-item-topline">
                    <span>{item.zone || "Unassigned"}</span>
                    <span>{item.category}</span>
                  </div>
                  <button
                    className="remove-item-button"
                    type="button"
                    aria-label={`Remove ${item.title}`}
                    onClick={() => onRemoveItem(item.id)}
                  >
                    Remove
                  </button>
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

function printSchedule(items: ScheduleItem[], projectName: string) {
  const printWindow = window.open("", "_blank", "width=1100,height=800");

  if (!printWindow) {
    window.print();
    return;
  }

  printWindow.document.write(buildPrintDocument(items, projectName));
  printWindow.document.close();
  printWindow.focus();
  printWindow.print();
}

function buildPrintDocument(items: ScheduleItem[], projectName: string): string {
  const printedAt = new Date().toLocaleDateString();

  return `<!doctype html>
<html>
  <head>
    <meta charset="utf-8" />
    <title>${escapeHtml(projectName)}</title>
    <style>
      @page { margin: 0.45in; size: landscape; }
      * { box-sizing: border-box; }
      body {
        color: #000;
        font-family: Arial, Helvetica, sans-serif;
        font-size: 9pt;
        margin: 0;
      }
      header {
        border-bottom: 3px solid #000;
        margin-bottom: 12pt;
        padding-bottom: 9pt;
      }
      h1 {
        font-size: 20pt;
        line-height: 1.05;
        margin: 0 0 4pt;
      }
      .meta {
        font-size: 8pt;
        font-weight: 700;
      }
      table {
        border-collapse: collapse;
        width: 100%;
      }
      thead {
        display: table-header-group;
      }
      th {
        border-bottom: 2px solid #000;
        font-size: 7.5pt;
        padding: 4pt;
        text-align: left;
        text-transform: uppercase;
        vertical-align: bottom;
      }
      td {
        border-bottom: 1px solid #000;
        font-size: 8pt;
        line-height: 1.25;
        padding: 5pt 4pt;
        vertical-align: top;
      }
      tr {
        break-inside: avoid;
      }
      .thumb {
        border: 1px solid #000;
        height: 0.62in;
        width: 0.62in;
      }
      .thumb img {
        filter: grayscale(100%);
        height: 100%;
        object-fit: cover;
        width: 100%;
      }
      .no-image {
        align-items: center;
        display: flex;
        font-size: 6.5pt;
        font-weight: 800;
        height: 100%;
        justify-content: center;
        text-align: center;
        text-transform: uppercase;
      }
      .source {
        overflow-wrap: anywhere;
      }
    </style>
  </head>
  <body>
    <header>
      <h1>${escapeHtml(projectName)}</h1>
      <div class="meta">${items.length} ${items.length === 1 ? "item" : "items"} / Printed ${escapeHtml(printedAt)}</div>
    </header>
    <table>
      <thead>
        <tr>
          <th>Image</th>
          <th>Zone</th>
          <th>Category</th>
          <th>Item</th>
          <th>Manufacturer</th>
          <th>Model</th>
          <th>Finish</th>
          <th>Description / Notes</th>
          <th>Required Add-ons</th>
          <th>Source</th>
        </tr>
      </thead>
      <tbody>
        ${items.map(printRow).join("")}
      </tbody>
    </table>
  </body>
</html>`;
}

function printRow(item: ScheduleItem): string {
  return `<tr>
    <td>
      <div class="thumb">
        ${
          item.sourceImageUrl
            ? `<img src="${escapeAttribute(item.sourceImageUrl)}" alt="" />`
            : `<div class="no-image">No image</div>`
        }
      </div>
    </td>
    <td>${escapeHtml(item.zone || "Unassigned")}</td>
    <td>${escapeHtml(item.category)}</td>
    <td><strong>${escapeHtml(item.title)}</strong></td>
    <td>${escapeHtml(item.manufacturer || "-")}</td>
    <td>${escapeHtml(item.modelNumber || "-")}</td>
    <td>${escapeHtml(item.finish || "-")}</td>
    <td>${escapeHtml(item.description || "-")}</td>
    <td>${escapeHtml(item.requiredAddOns.join(", ") || "-")}</td>
    <td class="source">${escapeHtml(item.sourceUrl)}</td>
  </tr>`;
}

function escapeHtml(value: string): string {
  return value
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#039;");
}

function escapeAttribute(value: string): string {
  return escapeHtml(value).replace(/`/g, "&#096;");
}
