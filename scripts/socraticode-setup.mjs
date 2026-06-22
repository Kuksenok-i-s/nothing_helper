import { Client } from "@modelcontextprotocol/sdk/client/index.js";
import { StdioClientTransport } from "@modelcontextprotocol/sdk/client/stdio.js";
import { fileURLToPath } from "node:url";
import path from "node:path";

const projectPath = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");

const transport = new StdioClientTransport({
  command: "npx",
  args: ["-y", "socraticode"],
  env: { ...process.env },
  cwd: projectPath,
});

const client = new Client({ name: "socraticode-setup", version: "1.0.0" }, { capabilities: {} });
await client.connect(transport);

async function call(name, args = {}) {
  const result = await client.callTool({ name, arguments: args });
  const text = result.content?.map((c) => c.text).filter(Boolean).join("\n") ?? JSON.stringify(result);
  console.log(`\n=== ${name} ===\n${text}`);
  return text;
}

function isComplete(text) {
  return /Indexing complete|✓ Indexing complete|Status: indexed/i.test(text) &&
    !/in progress|Progress:\s*\d+\/\d+ files/i.test(text);
}

function isInProgress(text) {
  return /in progress|Phase:|Progress:/i.test(text);
}

await call("codebase_index", { projectPath });

for (let i = 0; i < 120; i++) {
  const text = await call("codebase_status", { projectPath });
  if (isComplete(text)) break;
  if (!isInProgress(text) && /Indexed chunks:\s*[1-9]/i.test(text)) break;
  await new Promise((r) => setTimeout(r, 10000));
}

await client.close();
