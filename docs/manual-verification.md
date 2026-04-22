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
6. Edit at least `Zone` and `Title`.
7. Click `Undo`.
8. Confirm the generated draft values return.
9. Edit a field again.
10. Click `OK`.
11. Confirm the panel disappears.
12. Confirm the item count increments.
13. Refresh the page.
14. Confirm the item count remains.

## Known PoC Limits

- Extraction is mocked.
- Capture quality varies by page.
- Accepted items are local to the current Chrome profile and extension install.
- There is no schedule browser or Mothership dashboard yet.
- There is no product-page detection or context menu invocation yet.
