import { writeFile } from "node:fs/promises";
import { join } from "node:path";

const CONFIG_FILENAME = "hashbox_config.json";

export async function configureHashBox(token: string): Promise<string> {
  if (!token || !token.startsWith("HB-")) {
    throw new Error(
      "Invalid token: must be a non-empty string with HB- prefix"
    );
  }

  const configPath = join(process.cwd(), CONFIG_FILENAME);
  await writeFile(configPath, JSON.stringify({ token }, null, 2), "utf-8");

  return `HashBox configured successfully. Config saved to ${CONFIG_FILENAME}`;
}
