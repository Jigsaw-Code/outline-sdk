import { exec } from "node:child_process";
import { glob } from "glob";
import { promisify } from "node:util";
import archiver from "archiver";
import fs from "node:fs";
import handlebars from "handlebars";
import path from "node:path";
import minimist from "minimist";

import {
  OUTPUT_DIR,
  SDK_MOBILEPROXY_OUTPUT_DIR,
  WRAPPER_APP_OUTPUT_DIR,
  WRAPPER_APP_OUTPUT_SDK_MOBILEPROXY_DIR,
  WRAPPER_APP_OUTPUT_ZIP,
  WRAPPER_APP_TEMPLATE_DIR,
} from "./_constants.mjs";

export default async function main(
  {
    platform,
    entryDomain = "www.example.com",
    navigationUrl,
    additionalDomains = [],
  },
) {
  const sourceFilepaths = await glob(
    path.join(WRAPPER_APP_TEMPLATE_DIR, "**", "*"),
    {
      nodir: true,
      dot: true,
    },
  );

  for (const sourceFilepath of sourceFilepaths) {
    const destinationFilepath = path.join(
      OUTPUT_DIR,
      path.relative(
        WRAPPER_APP_OUTPUT_DIR,
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
        entryDomain,
        navigationUrl: navigationUrl ? navigationUrl : `https://${entryDomain}/`,
        additionalDomains,
      }),
      "utf8",
    );
  }

  if (!fs.existsSync(SDK_MOBILEPROXY_OUTPUT_DIR)) {
    await promisify(exec)(`npm run build:sdk_mobileproxy ${platform}`)
  }

  fs.mvSync(
    SDK_MOBILEPROXY_OUTPUT_DIR,
    WRAPPER_APP_OUTPUT_SDK_MOBILEPROXY_DIR,
  );

  await promisify(exec)(`
    cd ${WRAPPER_APP_OUTPUT_DIR}
    npm ci
    npx cap sync ${platform}
  `);

  await zip(WRAPPER_APP_OUTPUT_DIR, WRAPPER_APP_OUTPUT_ZIP);
}

function zip(root, destination) {
  const job = archiver("zip", { zlib: { level: 9 } });
  const destinationStream = fs.createWriteStream(destination);

  return new Promise((resolve, reject) => {
    job.directory(root, false);
    job.pipe(destinationStream);

    destinationStream.on("close", resolve);

    job.on("error", reject);
    destinationStream.on("error", reject);

    job.finalize();
  });
}

if (import.meta.url.endsWith(process.argv[1])) {
  const args = minimist(process.argv.slice(2));

  main(args).catch(console.error);
}
