# Sally V1 — Developer Handoff

## What this is

Sally V1 is a browser-based spec-capture concept for architects.

The core idea:

- user is on a product page
- clicks SPEC
- an embedded AI assistant (Sally) reads the page and proposes a structured spec row
- user edits the proposal
- clicks OK
- a returned schedule line appears for one more review
- clicks OK again
- the tool disappears and the user goes back to browsing

This is intentionally AI-with-human-custody, not AI replacing judgment.

## Current state

The existing canvas code is a front-end behavior prototype.

It currently demonstrates:

resting state with only the green SPEC button
temporary Sally panel
short AI-thinking state
editable Sally pass
reciprocal finish/model behavior
returned editable schedule line
undo in Sally and in returned review
return to browsing state after final OK

It is not yet:

a browser extension
connected to a real AI API
connected to a database
connected to persistent projects/schedules

## Product philosophy

Sally should not do things for people.
Sally should do things with people.

Machine responsibilities:

read messy product pages at scale
cull useful technical information
inspect images for clues
suggest likely required add-on items
propose a first draft

Human responsibilities:

judge
correct
fix intention
take custody
remain responsible

## Intended real-world architecture

1. Browser extension
Two invocation methods:

green SPEC toolbar button

browser context menu: SPEC this page

For V1, both do the exact same thing.

2. Sally pass
The extension captures page content and sends it to an AI extraction layer.

3. Mother Ship
Projects, schedules, editing, printing, and later spec-book output live here.

## What SPEC should capture from a page

V1 target payload:

page title
visible page text
page URL
main product image
structured product data if available
nearby cut-sheet/spec-sheet PDF links if found

Sally should extract:

item title
manufacturer
model number
category
description (AI synopsis favoring dimensions and installation constraints)
available finishes
finish-to-model mapping
likely required add-ons
likely optional companions

Examples of likely required add-ons:

toilet seat
pressure-balance valve
diverter
drain assembly
trap
rough valve body

## UX rules

### Sally panel

temporary
narrow ride-along panel
appears only when SPEC is invoked
editable before confirmation
visible Undo
disappears on OK

### Returned schedule review

appears after Sally OK
editable
visible Undo
disappears on OK

### Resting state

When the loop is done, user should see only:

the product page
the green SPEC button
current project context

## Mother Ship (next phase)

Project Home should contain:

project name
address
rooms/zones
schedules
print whole spec book on landing page
print this schedule on schedule subpages

Everything should remain hand-editable. No modes. No save button. Undo instead.

## Output philosophy

Printed output should be:

clean
bold
highly legible on site
still readable when dirty, folded, or slightly crumpled
Not precious.
Not over-designed.

TODO find an actual or realistic schedule

Swiss discipline is welcome, but field legibility matters more than graphic purity.

## Suggested repo structure

A reasonable first repo might look like this:

```
sally-v1/
  src/
    App.jsx
    components/
      SallyPanel.jsx
      ReviewRow.jsx
      ProductPageMock.jsx
      UndoButton.jsx
    lib/
      finishMapping.js
      productParsing.js
  public/
  package.json
  README.md
```

If moving toward the real product:

```
sally-v1/
  extension/
    manifest.json
    background.js
    contentScript.js
  web/
    src/
      App.jsx
      components/...
  server/
    api/...
```

## What to build next

### Immediate next technical step

Turn the current prototype into:

a clean React front end
a browser extension wrapper
a mocked AI extraction function
simple local persistence for projects/schedules

After that

replace mocked extraction with real model call
store projects and schedules in backend
build Mother Ship navigation and printing

## Important design constraints

Please preserve these:

SPEC is the invocation term, not Pin
green SPEC button
no hover behavior required for V1
right-click is just another way to invoke SPEC
no extra modes
no save button
undo should be obvious
everything user-facing remains editable
AI proposes; human approves

## Note for developer

The current canvas prototype is meant to communicate behavior and product feel, not final architecture.

Please treat the current UI and flow as the source of truth for interaction, while improving structure and reliability under the hood.
