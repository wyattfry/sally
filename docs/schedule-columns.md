# Schedule Columns

Most schedules will have, or start with, the same columns:
1. Code / Locator
2. Manufacterer
3. Model
4. Finish
5. Notes

While this project is in its early stages, when a schedule is created, it should have these columns. For schedules with a different schema, e.g. paint, insulation, it is acceptable to start with the defaults, then let the user add / remove / change them.

Here's a hypothetical user story with default columns:

1. A user with the extension installed, signed in, with a new empty project
2. User goes to product detail page for a light fixture, clicks SPEC
3. sally panel opens, "scanning...", it sees there are no schedules, proposes making a new one for electrical fixtures (not just light, can include fans, etc)
4. the electrical fixture schedule is created, and automatically has the five default columns.
5. the extractor IDs the right values for each for each field (except CODE)
6. CODE is computed: are there items already? no, so number will be "1". compute prefix: is "E" available? yes. Final proposed item code: "E-1"
7. the sally panel shows all five the fields filled in and asks user to approve adding it to the schedule or to cancel.

Non-default columns:

1. A user with the extension installed, signed in, with a new empty project
2. User goes to product detail page for a window, clicks SPEC
3. sally panel opens, "scanning...", it sees there is a Windows schedule with 2 items and includes its columns in the extraction request to the LLM
5. the extractor IDs the right values for each for each field (except CODE)
6. CODE is computed: are there items already? yes, so number will be 1 more than the highest (2 + 1 = 3). compute prefix: the schedule is using "W". Final proposed item code: "W-3"
7. the sally panel shows all five the fields filled in and asks user to approve adding it to the schedule or to cancel.