
# LLM Strategy

Here follows my conversation with Codex about improving the LLM strategy.

› i have this app i'm working on in ~/sally/ -- it scrapes a product detail webpage, sends it to an LLM with a requested JSON schema response, has the LLM extract the data, return the
  JSON, and uses that to essentially compile a spreadsheet / table / database. I've tried different services and models with varying results. Are there alternatives to the one-shot LLM
  prompt that could result in higher quality extraction? idk if making a custom AI agent would help, i don't really even know what that means... an example, it's been having a
  difficult time extracting values from a product page that has collapsed "Tech Specs", perhaps due to a character limit, I'm not sure. Advise.


• I’ll use the using-superpowers skill for the session rules, and brainstorming because this is an architecture/design question rather than a code edit.

• Explored
  └ Read SKILL.md (superpowers:using-superpowers skill), SKILL.md (superpowers:brainstorming skill)

─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

• I’m going to ground this in your existing extractor path first, so the advice maps to what Sally already does instead of staying abstract.

• Explored
  └ Search extract|json_schema|capturePage|Tech Specs|schema|Responses|chatcompletion in server
    Search README.md in sally

• Ran git -C /home/wyatt/sally log --oneline -5
  └ 1e065cc fix: live-update column header when renamed in modal
    014f615 Version update [skip actions]
    bbe7d7e feat: extraction detail page, zone sanitization, collaborators modal
    cd53ee2 Version update [skip actions]
    9b47937 feat: shared projects, column reorder, provider error hardening

• Explored
  └ Search extract|json_schema|capturePage|Tech Specs|schema|Responses|chatcompletion in server

• Explored
  └ Search README.md in sally

─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

• I found the relevant shape: extension DOM capture feeds a synchronous /api/v1/extract-spec, and the server already has provider adapters plus extraction logging. I’m checking the
  actual capture/prompt boundaries now, since collapsed spec failures are often input quality problems before they are model problems.

• Explored
  └ Read capturePage.ts, openai.go, types.go

─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

• Yes. A custom “agent” might help later, but I would not start there. Your bigger quality lever is changing Sally from “send whatever visible text fits into one prompt” to a staged
  extraction pipeline.

  The immediate issue is likely in your capture strategy. In /home/wyatt/sally/src/lib/capturePage.ts:3, Sally caps page text at 12000 chars. Collapsed content is only appended if
  visible text is under that cap, then the combined string is sliced again at 12000 chars: /home/wyatt/sally/src/lib/capturePage.ts:41. So “Tech Specs” can easily be omitted or
  truncated even before the LLM sees it.

  Recommended path:

  1. Improve deterministic page capture first
     Extract likely spec blocks separately from general visible text: accordions, tabs, aria-controls, hidden, display:none, data-*, product JSON, variant JSON, tables, definition
     lists, and embedded ecommerce state like __NEXT_DATA__, Shopify product JSON, etc. Send those as separate fields, not buried at the end of visibleText.
  2. Rank and compress before calling the model
     Build a small “evidence packet”:
      - product title
      - manufacturer / brand candidates
      - model / SKU candidates
      - spec tables
      - finish / variant tables
      - add-on/accessory sections
      - PDFs
      - structured data

     This can be mostly rule-based. The LLM should receive a curated packet, not a raw page dump.
  3. Use a two-pass LLM flow
     First call: identify relevant evidence spans and missing fields.
     Second call: produce the final strict JSON using only those spans.

     This usually beats one-shot extraction because the model has to “read” before it “fill forms.”
  4. Add field-level confidence and citations
     For each extracted value, ask for source evidence like "sourceText" or "sourceSection". You do not need to expose this to users, but it makes debugging much easier in your existing
     extraction logs/admin view.
  5. Fetch and parse PDFs selectively
     Your payload already includes pdfLinks, but it looks like you only pass links to the model. For architectural products, cut sheets often have the actual dimensions, finish codes,
     electrical data, etc. A server-side PDF text extraction pass for likely spec/cut-sheet PDFs would improve quality a lot.
  6. Use agents only for bounded tool use
     In this context, an “agent” just means an LLM loop that can choose actions: inspect DOM sections, fetch PDF, search within captured text, ask for more page data, then return JSON.
     That can help, but only after you have useful tools. Without better capture tools, an agent is just a more expensive one-shot prompt.

  My recommendation: implement deterministic capture + evidence packet + final structured extraction before building a full agent. It is simpler, cheaper, easier to test, and directly
  addresses the collapsed “Tech Specs” failure. The first concrete change I’d make is to stop treating collapsed specs as overflow text and add explicit request fields like
  specSections, tables, variantOptions, and documentText.