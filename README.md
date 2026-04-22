# Sally & Mothership

Initial design developer thoughts:

- prioritize creating a usable application quickly to aid short iterations
- use popular, mature stacks that make developing painless, e.g. QoL features like file watching, auto restarting server, ci/cd automation (maybe GitHub self-hosted agents?)
- include an auto-update mechanism in Chrome extension so end user need not perform awkward tasks whenever a change is published
- use strict dir / file structure
- use test-driven development
- keep functions pure, single-purpose
- consider using Docker Compose for initial PoC
- initial components: Chrome browser extension, Mothership dashboard application, database
- support multiple users, each user can have multiple projects, each project has one "schedule" (need to confirm), each entity with the full CRUD abilities
- use oauth or something else easy for first PoC
- select an appropriate AI agent to use in the loop to support two users, can be a service if free/included/cheap, or can be run on my own hardware
- for frontend, i've liked using Next.JS before for its completeness and dev-friendliness, but am open to whatever the standard is for extension development
- for backend, i prefer Go, keep things simple and clean. Makefiles.
- monorepo unless there's a compelling reason to go multi-repo

## Current PoC

This repo now contains a Chrome MV3 extension proof of concept for the Sally SPEC capture loop.

The PoC does this:

- injects an always-present green `SPEC` button onto normal webpages
- captures the current page title, URL, visible text, structured product JSON-LD, likely product image, and likely PDF/spec links
- uses a mocked extraction function to propose one editable schedule item
- lets the user edit the proposal, undo back to Sally's generated draft, cancel, or click `OK`
- saves accepted items to `chrome.storage.local`
- shows a small local item count next to the `SPEC` button

The PoC does not include backend storage, auth, a full Mothership dashboard, product-page detection, context menu invocation, or a real AI call.

## Development

Install dependencies:

```bash
npm install
```

Run tests:

```bash
npm test
```

Build the extension:

```bash
npm run build
```

Load the extension in Chrome:

1. Open `chrome://extensions`.
2. Enable Developer mode.
3. Click Load unpacked.
4. Select the generated `dist/` directory.
5. Open or refresh a normal `http://` or `https://` webpage.
6. Click the green `SPEC` button.

The extension stores accepted PoC items locally in the current Chrome profile through `chrome.storage.local`.
