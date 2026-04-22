# Manual Verification

## Chrome Extension Load

1. Run `npm run build`.
2. Open `chrome://extensions`.
3. Enable Developer mode.
4. Click Load unpacked.
5. Select `/home/wyatt/sally/dist`.
6. Open or refresh a normal `http://` or `https://` product page.

## PoC Flow

1. Confirm the green `SPEC` button appears with `Sally PoC` project context.
2. Click `SPEC`.
3. Confirm Sally opens as a right-side ride-along panel.
4. Confirm the panel briefly shows `Reading page`.
5. Confirm editable proposal fields appear.
6. Pick an existing `Zone`.
7. Choose `Add new zone...`, add a new zone, and confirm the new zone is selected.
8. Edit at least `Title`.
9. Press `Esc`.
10. Confirm the panel minimizes and a `Restore Sally draft` button appears.
11. Restore the draft and confirm edits remain.
12. Click `OK`.
13. Confirm the panel disappears.
14. Confirm the item count increments and the floating control shows `Page spec'd`.
15. Click the `Sally PoC` / item count context.
16. Confirm the captured schedule drawer opens and shows the accepted item with a thumbnail.
17. Close the drawer.
18. Refresh the page.
19. Confirm the item count remains and the page still shows as spec'd.

## Known PoC Limits

- Extraction is mocked.
- Capture quality varies by page.
- Accepted items are local to the current Chrome profile and extension install.
- There is no schedule browser or Mothership dashboard yet.
- There is no product-page detection or context menu invocation yet.
