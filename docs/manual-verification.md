# Manual Verification

## Chrome Extension Load

1. Run `npm run build`.
2. Open `chrome://extensions`.
3. Enable Developer mode.
4. Click Load unpacked.
5. Select `/home/wyatt/sally/dist`.
6. Open or refresh a normal `http://` or `https://` product page.

## PoC Flow

1. Confirm only the green `SPEC` button appears.
2. Click `SPEC`.
3. Confirm Sally opens as a right-side ride-along panel.
4. Confirm the panel briefly shows `Reading page`.
5. Confirm the panel header shows the project name, initially `My New Project`.
6. Confirm editable proposal fields appear.
7. Pick an existing `Zone`.
8. Choose `Add new zone...`, add a new zone, and confirm the new zone is selected.
9. Pick one of the default `Category` options.
10. Edit at least `Title`.
11. Press `Esc`.
12. Confirm the panel minimizes and a `Restore Sally draft` button appears.
13. Restore the draft and confirm edits remain.
14. Click `OK`.
15. Confirm the panel disappears, the `SPEC` button remains visually unchanged, and an `Item added` toast appears with a progress bar.
16. Click `SPEC` again.
17. Click `View Items` in the Sally panel.
18. Confirm the add-item panel closes and the captured schedule drawer opens in front with the accepted item and a thumbnail.
19. Confirm the project title shows a small edit affordance, click `My New Project`, rename it, and confirm the drawer title updates.
20. Click `Print`.
21. Confirm a new print tab/window opens with only the schedule, the renamed project title, and the browser print dialog opens.
22. Confirm the preview is a clean schedule sheet without the original product page behind it.
23. Close the print dialog and print tab/window.
24. Hover the accepted item and click `Remove`.
25. Confirm the item disappears and the viewer shows the empty state.
26. Close the drawer.
27. Refresh the page.
28. Confirm the `SPEC` button still appears in its normal green resting state.

## Context Menu

1. Right-click a normal webpage.
2. Click `SPEC this page`.
3. Confirm Sally opens and follows the same flow as clicking the green `SPEC` button.

## Known PoC Limits

- Extraction is mocked.
- Capture quality varies by page.
- Accepted items are local to the current Chrome profile and extension install.
- There is no full Mothership dashboard yet.
- There is no product-page detection yet.
