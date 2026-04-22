# Sally Extension Capture PoC Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a Chrome MV3 extension PoC that injects an always-present SPEC button, captures page data, shows one editable Sally proposal, and stores accepted items in `chrome.storage.local`.

**Architecture:** Use Vite to bundle a React content-script UI into `dist/`, with a static MV3 manifest copied from `public/`. The content script creates a Shadow DOM root, renders React inside it, and uses small library modules for capture, mock extraction, and storage.

**Tech Stack:** Vite, React, TypeScript, Vitest, Chrome MV3 extension APIs.

---

### Task 1: Project Scaffold

**Files:**
- Create: `package.json`
- Create: `index.html`
- Create: `tsconfig.json`
- Create: `tsconfig.node.json`
- Create: `vite.config.ts`
- Create: `public/manifest.json`
- Create: `src/contentScript.tsx`
- Create: `src/styles.css`
- Create: `src/vite-env.d.ts`

**Steps:**
1. Add Vite, React, TypeScript, and Vitest dependencies and scripts.
2. Configure Vite to build `src/contentScript.tsx` as `dist/contentScript.js`.
3. Add an MV3 manifest that loads `contentScript.js` on normal web pages and requests `storage` permission.
4. Add a minimal React content script that mounts into a Shadow DOM host.
5. Run `npm install`.
6. Run `npm run build` and confirm extension assets are emitted.

### Task 2: Domain Types And Storage

**Files:**
- Create: `src/lib/types.ts`
- Create: `src/lib/storage.ts`
- Create: `src/lib/storage.test.ts`

**Steps:**
1. Write Vitest tests for listing items, saving an item, and clearing items against a mocked `chrome.storage.local`.
2. Run the tests and confirm they fail because the storage module does not exist.
3. Implement `ScheduleItem`, `CapturedPage`, `listScheduleItems`, `saveScheduleItem`, and `clearScheduleItems`.
4. Run storage tests and confirm they pass.

### Task 3: Page Capture And Mock Extraction

**Files:**
- Create: `src/lib/capturePage.ts`
- Create: `src/lib/capturePage.test.ts`
- Create: `src/lib/mockExtraction.ts`
- Create: `src/lib/mockExtraction.test.ts`

**Steps:**
1. Write tests for capturing title, URL, visible text, JSON-LD product data, PDF/spec links, and a likely main image from a synthetic DOM.
2. Run capture tests and confirm they fail because capture is not implemented.
3. Implement `capturePage(document, window.location)`.
4. Run capture tests and confirm they pass.
5. Write tests for deterministic schedule proposal fields from a captured page.
6. Run extraction tests and confirm they fail because extraction is not implemented.
7. Implement `mockExtractScheduleItem`.
8. Run extraction tests and confirm they pass.

### Task 4: React UI Flow

**Files:**
- Create: `src/App.tsx`
- Create: `src/components/SpecButton.tsx`
- Create: `src/components/SallyPanel.tsx`
- Create: `src/components/ScheduleViewer.tsx`
- Modify: `src/contentScript.tsx`
- Modify: `src/styles.css`

**Steps:**
1. Write component tests for the core flow if the test setup remains light enough; otherwise keep logic in testable libraries and verify UI through build/manual loading.
2. Implement the always-present SPEC button.
3. Implement the Sally panel with thinking, editable fields, View Items, cancel, and OK.
4. Wire capture, mock extraction, storage save, toast feedback, and schedule viewer refresh.
5. Run `npm test`.
6. Run `npm run build`.

### Task 5: Documentation And Manual Verification

**Files:**
- Modify: `README.md`
- Create: `docs/manual-verification.md`

**Steps:**
1. Document install/build/load-unpacked steps.
2. Document the manual PoC verification checklist.
3. Run `npm test`.
4. Run `npm run build`.
5. Report the commands and results.
