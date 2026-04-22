import "@testing-library/jest-dom/vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import App from "./App";

const storageState: Record<string, unknown> = {};

function installChromeStorageMock() {
  vi.stubGlobal("chrome", {
    storage: {
      local: {
        get: vi.fn(async (keys: string | string[]) => {
          const result: Record<string, unknown> = {};
          for (const key of Array.isArray(keys) ? keys : [keys]) {
            result[key] = storageState[key];
          }
          return result;
        }),
        set: vi.fn(async (values: Record<string, unknown>) => {
          Object.assign(storageState, values);
        }),
        remove: vi.fn(async (key: string) => {
          delete storageState[key];
        })
      }
    }
  });
}

describe("App", () => {
  beforeEach(() => {
    document.title = "Example Co. WF-200 Wall Faucet";
    document.head.innerHTML = `
      <meta property="og:image" content="https://example.com/faucet.jpg" />
      <script type="application/ld+json">
        {"@type":"Product","name":"Wall Faucet","brand":{"name":"Example Co."},"sku":"WF-200"}
      </script>
    `;
    document.body.innerHTML = `
      <p>Polished chrome wall-mounted faucet. Requires rough valve body.</p>
    `;
    for (const key of Object.keys(storageState)) {
      delete storageState[key];
    }
    installChromeStorageMock();
  });

  it("opens Sally, edits a proposal, saves it, and refreshes the item count", async () => {
    const user = userEvent.setup();
    render(<App />);

    expect(await screen.findByText("0 items")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "SPEC" }));

    expect(screen.getByText("Reading page")).toBeInTheDocument();
    expect(await screen.findByDisplayValue("Wall Faucet")).toBeInTheDocument();

    await user.clear(screen.getByLabelText("Zone"));
    await user.type(screen.getByLabelText("Zone"), "Primary Bath");
    await user.clear(screen.getByLabelText("Title"));
    await user.type(screen.getByLabelText("Title"), "Wall faucet revised");
    await user.click(screen.getByRole("button", { name: "OK" }));

    await waitFor(() => expect(screen.queryByText("Sally proposal")).not.toBeInTheDocument());
    expect(await screen.findByText("1 item")).toBeInTheDocument();
  });

  it("undo restores the generated proposal", async () => {
    const user = userEvent.setup();
    render(<App />);

    await user.click(await screen.findByRole("button", { name: "SPEC" }));
    const title = await screen.findByLabelText("Title");
    await user.clear(title);
    await user.type(title, "Changed title");

    await user.click(screen.getByRole("button", { name: "Undo" }));

    expect(screen.getByDisplayValue("Wall Faucet")).toBeInTheDocument();
  });
});
