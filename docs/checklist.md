
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

- [ ] restore some signal of an item being added, idk what the best UX is, maybe change SPEC button to say "Captured!" and a few seconds later, goes back to SPEC? Or something else? I haven't decided on a good way to offer the user a way to go from an ecommerce page to the mothership page
- [ ] pressing Enter in any field in the SallyPanel should submit the form
- [ ] update mothership web ui to use custom columns, support note-type schedules
- [ ] in sallypanel, project selection might feel better as text with a 'select different project' button that brings up a modal or something? it doesn't feel right as a combo box
- [ ] on projects/[id]: the "New Schedule [field] Add Schedule" fields / form is confusing, the create-schedule flow should use more conventional CRUD webapp UI
- [ ] see what happens if you try adding different finishes of a product, If the product page supports it, "finish" should be a combobox with two-way binding to MODEL
- [ ] schedule item tables: drag to re-order schedules, drag to re-order items, click headers to sort by, 
- [ ] built in blob storage for saving images so images don't have to be fetched from remotes every time
- [ ] breadcrumbs no longer needed, can be removed from views etc
- [ ] add link to chrome extension in mothership somewhere, maybe the footer? add a footer
- [ ] add link to mothership in chrome extension description
- [ ] user account page, dummy billing, stripe?
- [ ] how much to charge? how often? monthly or by use?
- [ ] make the View-Only mobile / tablet friendly


Hypothetical Flow for Extension:
1. get project with latest change, if none, make one?
2. does this item fit logically in an existing schedule? if not make one with some columns that would be common for that type of schedule
3. ask the LLM to return fields that match the columns of the selected schdule


schedyuels sohuld start with these default columsn:
1. locator number
    e.g. F-1, x-0 = owner, don't suffix a later
    windows can be W-1, W-2
    duplicate codes are unusual, default to incrementing
    doors can just be D-x
2. manufacturer
3. model number
4. finish
5. notes

contractor needs to accurately convey to hteir supplier on the phone
what they need

Some htings need accessories, but add this later