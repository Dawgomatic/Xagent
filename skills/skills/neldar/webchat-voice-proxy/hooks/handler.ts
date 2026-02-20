import { execSync } from "child_process";
import { join, dirname } from "path";
import { fileURLToPath } from "url";

const __dirname = dirname(fileURLToPath(import.meta.url));

const handler = async (event: any) => {
  if (event.type !== "gateway" || event.action !== "startup") return;

  const script = join(__dirname, "inject.sh");
  try {
    execSync(`bash "${script}"`, {
      timeout: 10_000,
      stdio: "pipe",
    });
    console.log("[voice-input-inject] Control UI patched successfully");
  } catch (err: any) {
    console.error("[voice-input-inject] Failed:", err.message ?? err);
  }
};

export default handler;
