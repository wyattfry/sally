# Sally Extension Capture PoC Design

## Goal

Build a proof of concept that validates the browser extension capture experience: an architect can invoke SPEC while browsing a real product page, review Sally's editable schedule proposal, accept it, and return to browsing.

## Scope

The PoC uses a real Chrome MV3 extension with a Vite and React injected UI. It has no backend, auth, database, product-page detection, context menu, or full Mothership dashboard. Persistence is limited to accepted schedule items in `chrome.storage.local`.

## User Flow

1. A small green `SPEC` button is always present on normal webpages.
2. The user clicks `SPEC`.
3. Sally opens a narrow ride-along panel and shows a short thinking state.
4. The content script captures page title, URL, visible text, candidate image, structured data, and likely PDF/spec links.
5. A mocked extraction function proposes one editable schedule item.
6. The user edits the proposal.
7. The user clicks `OK`.
8. The item is saved to `chrome.storage.local`.
9. Sally disappears and the user returns to browsing with only the `SPEC` button and small project/item-count context visible.

## Data Model

For the PoC, zones are fields on schedule items rather than parents of schedules:

```text
Project
  Schedule
    Item
      zone
```

The extension may store only an `items` array internally, but the item type should be compatible with later `Project -> Schedule -> Item` expansion.

## Architecture

The extension injects a content script into matching pages. The content script creates a Shadow DOM host for Sally so site styles do not leak into the UI and Sally styles do not leak into the page. React owns the injected UI state and calls small library modules for page capture, mock extraction, and extension storage.

The mock extractor should accept the same kind of captured payload a real extraction API will later receive. It should be deterministic and good enough to make the panel feel connected to the current page.

## Components

- `SpecButton`: always-visible green invocation control and item count.
- `SallyPanel`: thinking state and editable proposal form.
- `UndoButton`: restores the proposal to its last generated state during review.
- Capture library: reads the active page DOM.
- Mock extraction library: turns captured page data into a schedule item proposal.
- Storage library: wraps `chrome.storage.local` behind `listScheduleItems`, `saveScheduleItem`, and `clearScheduleItems`.

## Acceptance Criteria

- The extension builds into loadable Chrome MV3 assets.
- Loading the unpacked extension injects a green `SPEC` button on webpages.
- Clicking `SPEC` opens Sally without navigating away.
- Sally presents one editable schedule proposal after a thinking state.
- `OK` persists the accepted item to `chrome.storage.local`.
- The panel disappears after `OK`.
- Reloading or changing pages preserves the accepted item count.
- The implementation remains mocked and local-only.

