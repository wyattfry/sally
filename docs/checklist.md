
# Things to address

- [x] Add "notes" field to schedules, see ./docs/exmaple schedule.pdf > Appliance Schedule
- [x] Add ability to group items within a schedule, e.g. schedule: appliance schedule, sub-groups: wet bar, kitchen, heating/cooling, basement. This seems to be what the previous "Zone" field was describing. In the example project schedules PDF, some schedules group their items by room/zone with vertical text on the left side of the table. We don't need exactly that same layout, but perhaps the same concept.
- [x] if the LLM returns <UNKNOWN> for zone, translate that to the equivalent to the "No zone" option
- [x] Test the flow:
    1. made a new project and schedule in mothership
    2. go to new product page in browser, click "SPEC"
    3. observe, it did not default to using the newest project
    Is this because it only considers a project "updated" when an item is added? It should be **any** change, including making a new empty schedule, editing a note/commment, etc
- [x] projects/[id] should show the project description and thumbnail
- [x] i notice project_handlers.go is getting long, >1k lines, and contians section separator comments. Let's do a pass through the Go files for restructuring the directories / files into a more sustainable pattern, e.g. split large files along section separators
- [x] Come up with design for each user adding/removing columns and not overwhelming the database with migrations, is there already a way to do this?  Or do we do something janky like a 'custom columns' column with JSON or something?
- [x] add "empty" / custom schedules that are actually just for notes, not items, see PDF
- [x] get Google OAuth client ID / secret, wire up to local
- [x] add favicon for go server
- [x] ensure extension can auth
- [x] going to /login when already logged in should redirect to / or /projects
- [x] after clicking 'Sign in wiht google' in extension, instead of opening a new tab, could it open a pop up window? and when logged in, the extension self-refreshes? right now, it opens a tab insteda, and then goes to the mothership web app. I have to go back to the ecommerce site tab, where the extension still says 'login with google', and i have to refresh the page.
- [x] the CODE field should default to some auto-incrementing value, multiple items may have the same code, use ./docs/example schedule.pdf -- it seems to follow the pattern
    1. capital first letter of schedule name, and sometimes an additional letter from the name, e.g. Paint -> PT
    2. a dash
    3. an incrementing number, can start at 0, 1, or higher
    4. incrementing captial letters, starting at A
- [x] the mothership portal looks VERY DIFFERENT from the info / landing / annoucement teaser page at spexxtool.com --- can the portal look more like the info page?
- [x] re-implement default columns to:
    1. Code / Locator
    2. Manufacterer
    3. Model
    4. Finish
    5. Notes
- [x] on projects/[id]: the "New Schedule [field] Add Schedule" fields / form is confusing, the create-schedule flow should use more conventional CRUD webapp UI
- [x] add link to chrome extension in mothership somewhere, maybe the footer? add a footer
- [x] make the View-Only mobile / tablet friendly, maybe more card-like than table-like?
- [x] support note-type schedules
- [x] human friendly 404 page on mothership, e.g. if you try to go to a share link for a deleted project
- [x] project csv export?
- [x] add custom item to a schedule / manual entry, e.g. if no online page exists for the product, or if it is an owner-provided item
- [x] update mothership web ui to use custom columns
- [x] restore some signal of an item being added, idk what the best UX is, maybe change SPEC button to say "Captured!" and a few seconds later, goes back to SPEC? Or something else? I haven't decided on a good way to offer the user a way to go from an ecommerce page to the mothership page
- [x] built in storage for saving images so images don't have to be fetched from remotes every time. Maybe local dev has a bind mount, dev server too?
- [x] refine the "Add item" UX, it requires a page reload, the page jumps, the Add Item button awkwardly spans the entire width, the row is unexpectedly added in the highest location, not appended, as i was expecting, to the bottom. Maybe the empty fields could show a hint "Click to edit" as is done elsewhere on the site.

Critical Path
- [ ] bug: i had a project with a schedule full of insulation. i then went to a product page for a window, spec'd it, and it wanted to add it to the insulation schedule (the llm extraction should have decided that the item did not logically fit with the items in the insulation schedule). i opted to add it to a new schedule named 'Window Schedule', the fields in sallypanel correctly updated back to the default fields (no R-Value), but the CODE kept the "I-" prefix from the previous table, when it should have determined a new Code prefix and number. Investigate.
- [ ] bug: i had a project with a Windows Schedule with items. I went to a product page for a window, spec'd it, sallypanel wanted to make a new schedule called "Window Schedule [current selected]". 
- [ ] bug: project with windows schedule with items, i added custom columns: rough opening, overall jamb, swing. i spec'd a window product page, the custom fields were empty. the product page has a Specifications collapsable section with the desired data for the custom fields, check whether the specs were sent to the LLM, if so, investigate why the custom fields came back empty. product page: https://www.homedepot.com/p/Hy-Lite-47-5-in-x-11-5-in-Manchester-Silkscreened-Decorative-Glass-White-New-Construction-Frame-Window-DF4711MCSSWHV1500/331463170
- [ ] notes to support images, png / svg / copy-paste from CAD?
- [ ] add "CODE" label to the code in each item's tile to help it stand out, indicate the significance
- [ ] add link to mothership in chrome extension description
- [ ] add an About page that epxlains what problem this app solves, how to use it, how to install it, FAQs, etc
- [ ] collaborator / editor share link in addition to read only share link
- [ ] notes to support multiple "rows" or inner-sections?
- [ ] user account page, dummy billing, stripe?
- [ ] a feature to delete all of a users data / opt-out

Nice To Have
- [ ] chrome ext: easy way to toggle SPEC button visibility
- [ ] grafana / prometheus for admin view, showing users, activity, LLM calls, server load, storage, etc
- [ ] show LLM / token usage, maybe by day? week? in user profile / settings / account page
- [ ] update the homepage content to reflect the latest look and function of the site and ext
- [ ] make the project detail page's project name, address, desc narrower, atm they fill the available width, which is awkwardly wide. And their background color shold be slightly different than the page to show the user the clickable / editable area
- [ ] pressing Enter in any field in the SallyPanel should submit the form
- [ ] in sallypanel, project selection might feel better as text with a 'select different project' button that brings up a modal or something? it doesn't feel right as a combo box
- [ ] see what happens if you try adding different finishes of a product, If the product page supports it, "finish" should be a combobox with two-way binding to MODEL
- [ ] breadcrumbs no longer needed, can be removed from views etc
- [ ] alternative to google sign-in
- [ ] optimize LLM spend, both for development and data extraction
- [ ] would a 'duplicate an item' feature be useful?
- [ ] how much to charge? how often? monthly or by use?
- [ ] print view version of shared page?
