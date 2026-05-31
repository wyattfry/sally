
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
- [x] the CODE field should default to some auto-incrementing value, multiple items may have the same code, use ./docs/example schedule.pdf -- it seems to follow the patternANTHROPIC_MODEL=claude-haiku-4-5

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
- [x] grafana / prometheus for admin view, showing users, activity, LLM calls, server load, storage, etc
- [x] collaborator / editor share link in addition to read only share link
- [x] renaming a column requires a refresh to see update

Bugs
- [x] "extracted inappropriate value for zone": project with schedule with 0 items, spec'd a product, Zone field extracted value: `</zone><parameter name="suggestedScheduleName">Electrical Fixture Schedule` — sanitizeZone() strips XML artifacts; schema description + prompt reinforcement added
- [x] item detail modal does not scroll
- [x] it should prompt to make a project if none exist before extraction

- [x] some sites hide the SPEC button and panel
    - https://www.aspectled.com/products/9-led-large-in-ground-in-wall-led-light-rgb-white-36w
    - https://simplygoodcoffee.com/products/the-brewer-plastic-free
- [x] "[current selected]": this is still happening! investigate again. i had a project with a Windows Schedule with items. I went to a product page for a window, spec'd it, sallypanel wanted to make a new schedule called "Window Schedule [current selected]". 
- [x] "stale code prefix": project with a schedule with insulation items. spec'd a product page for a window, and it wanted to add it to the insulation schedule (the llm extraction should have decided that the item did not logically fit with the items in the insulation schedule). i opted to add it to a new schedule named 'Window Schedule', the fields in sallypanel correctly updated back to the default fields (no R-Value), but the CODE kept the "I-" prefix from the previous table, when it should have determined a new Code prefix and number. Investigate.
- [x] "empty custom fields": project with windows schedule with items, i added custom columns: rough opening, overall jamb, swing. i spec'd a window product page, the custom fields were empty. the product page has a Specifications collapsable section with the desired data for the custom fields, check whether the specs were sent to the LLM, if so, investigate why the custom fields came back empty. product page: https://www.homedepot.com/p/Hy-Lite-47-5-in-x-11-5-in-Manchester-Silkscreened-Decorative-Glass-White-New-Construction-Frame-Window-DF4711MCSSWHV1500/331463170
- [ ] "HTTPS only error": got raw error in extension: `provider failure: upstream status 400: {"type":"error","error":{"type":"invalid_request_error","message":"Only HTTPS URLs are supported."},"request_id":"req_011CamiCiV7uMRZHD25TAN8j"}`
- [ ] "Download failure error": `provider failure: upstream status 400: {"type":"error","error":{"type":"invalid_request_error","message":"Unable to download the file. Please verify the URL and try again."},"request_id":"req_011Camj7gwaZSXvzqSjAht7C"}`
- [ ] "robots.txt error": extension error: `provider failure: upstream status 400: {"type":"error","error":{"type":"invalid_request_error","message":"This URL is disallowed by the website's robots.txt file."},"request_id":"req_011CamkYhD1i1sJRXQ5rp8zt"}`
- [ ] project with appliance and paint schedules, spec'd paint, wanted to put it in the appliance schedule, wtf??
- [x] factor shared projects in selection of most recently changed project default
- [x] remove image picker from item detail modal
- [x] change "copy" to "copying" in the about pcage
- [x] warn if item is out of stock
- [ ] actions menu, remove bold from all items
- [ ] item "Details" button can persist, not just on hover
- [ ] sallpypanel project switcher isn't switching to shared projects
- [ ] in project detail view, switch from rows tiles / cards with 
- [ ] sallpy panel "edit columns" -> "edit"
- [ ] clicking 'View Items' launches mothership in popup, not new tab
- [ ] add hint to mothership via 'view projects' to tell user how to keep shopping
- [ ] sallypanel: make file attachments smaller, notes field bigger
- [ ] sallypanel, during extraction show some message
- [ ] lightology melt pendant, LLM got manufactorer as 'Tom Dixon' but that is the designer, the manf is 'Lightology'
- [ ] notes: LLM instructions: process the "notes" field, we are only interested in info interesting to architects and contractors, e.g. measurments, what the item is, finish, description but not commercial. Things not to include: warranty, certifications, country of origin
- [ ] share link: share/firstname.lastname/project-slug, warn that on project name change that old share links will break
- [ ] contractor view: clicking thumbnails should show the same picker as architect view
- [ ] download as DXF to import into drawing, or PDF booklet "project manual", cover page with project header, index, no page break between schedules, same columns as the contractor
- [ ] about: slide 2, say the button is GREEN. in the intro paragraph say you can also export dxf or pdf project manual
- [ ] google oauth api: update name or make a new one, atm it is "sally-ci"
- [ ] sally panel: if LLM server is unavailable, user sees "Connection to background script closed unexpectedly." -- replace with something better
- [ ] google auth: handle if user has cookies turned off (private mode?)
- [ ] optimize extraction: build mapping / profile by ecommerce site for where to find which fields on the page
- [ ] mothership: add way to remove oneself from a shared project

Critical Path
- [x] add an About page that epxlains what problem this app solves, how to use it, how to install it, FAQs, etc
- [x] re-order schedules option in the Actions menu, modal that looks similar to the Edit Columns modal (up / down buttons, but no rename or delete. ideally don't need a page reload to see updates)
- [x] sallypanel: if new schedule, give user a way to add/remove/edit the columns, if not every time
- [x] what about LLM cacheing? the responses from anthropic mention it:
    ```json
    "usage": {
        "cache_creation": {
            "ephemeral_1h_input_tokens": 0,
            "ephemeral_5m_input_tokens": 0
        },
        "cache_creation_input_tokens": 0,
        "cache_read_input_tokens": 0,
    }
    ```
- [x] sally panel: replace progress bar countdown to close the panel after adding an item. the bar is vague as to what it represents. Instead, let's reomve the bar, and instead add another button under 'View Project' that says 'Close Panel Now', with maybe a little label under it that says 'Closing panel in [x] seconds...'? But what if they user wanted to keep the panel open? Consider the non-tech savvy user that needs more explicit prompting, less familiar with UX paradigms. Maybe something like 'Item has been added to your project / schedule, you may now close this panel. To view your project after this panel closes, [instructions]' whatever those steps may be, e.g. right click > sally > view schedule
- [x] ALWAYS capture procurement information for contractor: price, lead time, shipping speed, shipping cost, out of stock, back ordered. A contractor might pay more overall for a shorter lead time, but they also want to stay within budget
- [ ] notes to support images, png / svg / copy-paste from CAD?
- [ ] add link to mothership in chrome extension description
- [ ] a feature to delete all of a users data / opt-out
- [x] user account page, dummy billing, stripe?
- [ ] defend against out of control infra costs / abuse
- [ ] research cloudflare json extractor: https://developers.cloudflare.com/browser-run/quick-actions/json-endpoint/
- [ ] notes to support multiple "rows" or inner-sections?
- [ ] provision prod environment
- [x] contractor view: clicking thumbnails should show a modal with a larger rendering of the image

Nice To Have
- [x] LLM: scrape PDF content
- [x] LLM: few-shot examples
- [x] update the homepage content to reflect the latest look and function of the site and ext
- [x] admin: in the extraction calls table, add a request ID column, clicking will show you a detail page of that request, including sanitized request and repsonse, status, body, etc
- [x] admin: create test user, magic-link
- [x] in sallypanel and mothership, capture all the product images and allow user to choose which one to use, or upload a new one
- [x] fix jupyter notebook json error in 'collect logs'
- [x] don't allow schedules with duplicate names
- [x] admin: line graphs dont seem to work. Use a library? D3.js?

- [x] create a report of the expenses estimated for scaling up to the first milestone of users, e.g. 100? estimate the cost for cloud compute spend, database, LLM, what else? network traffic? BC/DR? Estimate what users would have to be charged to cover costs.
- [-] add "CODE" label to the code in each item's tile to help it stand out, indicate the significance
- [ ] mothership: add hint text while editing a field on how to save and cancel
- [-] add Swagger: https://github.com/swaggo/swag
- [ ] more detail while extraction is underway, show the streaming data? or show the fields but with some animation instead of text values?
- [ ] move an item between schedules
- [ ] add a 'report bug / feedback' button on the mothership, send messages that arevisible in the admin portal
- [ ] admin: use google user group?
- [ ] alternatives to one-shot general purpose LLM? can i "build an agent" w/e that means?
- [-] admin: paginate tables (users, extraction calls)
- [ ] chrome ext: easy way to toggle SPEC button visibility
- [x] show LLM / token usage, maybe by day? week? in user profile / settings / account page
- [x] make the project detail page's project name, address, desc narrower, atm they fill the available width, which is awkwardly wide. And their background color shold be slightly different than the page to show the user the clickable / editable area
- [ ] pressing Enter in any field in the SallyPanel should submit the form
- [ ] in sallypanel, project selection might feel better as text with a 'select different project' button that brings up a modal or something? it doesn't feel right as a combo box
- [x] see what happens if you try adding different finishes of a product, If the product page supports it, "finish" should be a combobox with two-way binding to MODEL
- [-] breadcrumbs no longer needed, can be removed from views etc
- [ ] alternative to google sign-in
- [ ] optimize LLM spend, both for development and data extraction
- [ ] would a 'duplicate an item' feature be useful?
- [x] how much to charge? how often? monthly or by use?
- [ ] print view version of shared page?
