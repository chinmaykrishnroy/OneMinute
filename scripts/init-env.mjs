import { existsSync, readFileSync, writeFileSync } from "node:fs";
import { randomBytes } from "node:crypto";
if (existsSync(".env")) {
  console.log(".env already exists; preserved.");
} else {
  let contents = readFileSync(".env.example", "utf8");
  for (const key of ["POSTGRES_PASSWORD", "REDIS_PASSWORD", "TURN_SECRET"]) {
    contents = contents.replace(new RegExp("^" + key + "=.*$", "m"), key + "=" + randomBytes(32).toString("hex"));
  }
  writeFileSync(".env", contents, { mode: 0o600, flag: "wx" });
  console.log("Created .env with unique local secrets.");
}
