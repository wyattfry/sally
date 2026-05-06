# The "Code" aka "Locator" Item Field

Items have a field called "Code". The purpose of this field is to assign a short identifier to the item. The identifier is to be (mostly) unique within the context of the project. This idenifier is used in drawings to concisely indicate which item goes in which **location**, as opposed to writing out the entire make and model etc (the field may alternatively be called "locator"). For example, a contractor at the construction site would look at the drawing, see a code, then refer to the schedule to correlate which item is indicated. Therefore, the most important quality of the codes is that they are **easily distinguishable**.

There is no industry standard for how they are derived, and can vary between architects. Here are some guidelines that can provide a starting place for using them:

1. The values are usually in the format "[letter]-[number]"
2. The "letter" portion is usually one uppercase character that is easily relatable back to its containing schedule. If two schedules have names that share a starting letter, a second letter may be chosen to make the two referred-to schedules easier to distinguish.
3. The numbering usually starts at 1 and increments by 1 for each item added. If an item is added or removed in the middle of a list, we want to be careful about not automatically changing the codes of the subsequent rows, as this value is an **identifier**, not just an index
4. There may be edge cases where the same code is used for multiple items (such as all owner-provided items using "0"), but the majority of the cases will have unique codes for each item. The app should default to unique codes, but not stop the user from re-using codes.
5. Non-schedule tables, e.g. notes, are not required have a code column