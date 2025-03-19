import handlebars from "handlebars";
import fs from "node:fs";
import path from "node:path";
import { glob } from "glob";

const INPUT_DIR = "wrapper_template";
const OUTPUT_DIR = "output/wrapper_app";

export async function main(
  {
    mainDomain = process.env.TARGET_DOMAIN || "www.example.com",
    navigationUrl,
    whitelistedDomains = process.env.WHITELISTED_DOMAINS?.split(/,\s*/) ?? [],
  },
) {
  const sourceFilepaths = await glob(
    path.join(process.cwd(), INPUT_DIR, "**", "*"),
    {
      nodir: true,
      dot: true,
    },
  );

  for (const sourceFilepath of sourceFilepaths) {
    const destinationFilepath = path.join(
      process.cwd(),
      OUTPUT_DIR,
      path.relative(
        path.join(process.cwd(), INPUT_DIR),
        sourceFilepath,
      ),
    );

    // ensure directory
    fs.mkdirSync(path.dirname(destinationFilepath), { recursive: true });

    if (!sourceFilepath.endsWith(".handlebars")) {
      fs.copyFileSync(sourceFilepath, destinationFilepath);
      continue;
    }

    const template = handlebars.compile(
      fs.readFileSync(sourceFilepath, "utf8"),
    );

    fs.writeFileSync(
      destinationFilepath.replace(/\.handlebars$/, ""),
      template({
        mainDomain,
        navigationUrl: navigationUrl ? navigationUrl : `https://${mainDomain}/`,
        whitelistedDomains,
      }),
      "utf8",
    );

    fs.mvSync(
      path.join(process.cwd(), "ouput/mobileproxy"),
      path.join(process.cwd(), OUTPUT_DIR, "mobileproxy")
    );

    // TODO: zip output/wrapper_app
  }
}

if (import.meta.url.endsWith(process.argv[1])) {
  main().catch(console.error);
}
