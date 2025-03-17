import handlebars from "handlebars";
import fs from "fs";
import path from "path";
import { glob } from "glob";

// TODO: Replace with SED
const INPUT_DIR = "wrapper_template";
const OUTPUT_DIR = "output/wrapper_app";

if (!process.env.NGROK_DOMAIN) {
  throw new Error("NGROK_DOMAIN must be set!");
}

(async function () {
  const sourceFilepaths = await glob(path.join(process.cwd(), INPUT_DIR, "**", "*"), {
    nodir: true
  });

  for (const sourceFilepath of sourceFilepaths) {
    const destinationFilepath = path.join(
      process.cwd(),
      OUTPUT_DIR,
      path.relative(
        path.join(process.cwd(), INPUT_DIR),
        sourceFilepath
      )
    );

    // ensure directory
    fs.mkdirSync(path.dirname(destinationFilepath), { recursive: true });

    if (!sourceFilepath.endsWith(".handlebars")) {
      fs.copyFileSync(sourceFilepath, destinationFilepath);
      continue;
    }

    const template = handlebars.compile(fs.readFileSync(sourceFilepath, "utf8"));

    fs.writeFileSync(
      destinationFilepath.replace(/\.handlebars$/, ""),
      template({
        domain: process.env.NGROK_DOMAIN
      }),
      "utf8"
    );
  }
})();
