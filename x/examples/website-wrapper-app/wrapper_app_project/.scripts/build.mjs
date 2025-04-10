// Copyright 2025 The Outline Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import { exec } from "node:child_process";
import { glob } from "glob";
import { promisify } from "node:util";
import chalk from "chalk";
import archiver from "archiver";
import fs from "node:fs";
import handlebars from "handlebars";
import path from "node:path";
import minimist from "minimist";

const OUTPUT_DIR = path.join(process.cwd(), "output");
const WRAPPER_APP_TEMPLATE_DIR = path.join(
  process.cwd(),
  "wrapper_app_project/template",
);

const DEFAULT_SMART_DIALER_CONFIG = {
  dns: [{
    https: { name: "9.9.9.9" },
  }],
  tls: ["", "split:1", "split:2", "tlsfrag:1"],
};

export default async function main(
  {
    platform,
    sdkVersion = "x/v0.0.1",
    additionalDomains = [],
    appId,
    appName,
    entryUrl = "https://www.example.com",
    navigationUrl,
    output = OUTPUT_DIR,
    platform,
    smartDialerConfig = DEFAULT_SMART_DIALER_CONFIG,
  },
) {
  const WRAPPER_APP_OUTPUT_DIR = path.resolve(output, "wrapper_app_project");
  const WRAPPER_APP_OUTPUT_ZIP = path.resolve(output, "wrapper_app_project.zip");
  
  const SDK_MOBILEPROXY_OUTPUT_DIR = path.resolve(output, "mobileproxy");
  const WRAPPER_APP_OUTPUT_SDK_MOBILEPROXY_DIR = path.resolve(
    WRAPPER_APP_OUTPUT_DIR,
    "mobileproxy",
  );

  if (!fs.existsSync(SDK_MOBILEPROXY_OUTPUT_DIR)) {
    console.log(`Building the Outline SDK mobileproxy library for ${platform}...`);

    await promisify(exec)(`npm run build:mobileproxy ${platform} ${sdkVersion} ${output}`);
  }

  const sourceFilepaths = await glob(
    path.join(WRAPPER_APP_TEMPLATE_DIR, "**", "*"),
    {
      nodir: true,
      dot: true,
    },
  );

  console.log("Building project from template...");

  for (const sourceFilepath of sourceFilepaths) {
    const destinationFilepath = path.join(
      WRAPPER_APP_OUTPUT_DIR,
      path.relative(
        WRAPPER_APP_TEMPLATE_DIR,
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

    if (typeof additionalDomains === "string") {
      additionalDomains = [additionalDomains];
    }

    const entryUrlParts = new URL(entryUrl);

    const entryDomain = entryUrlParts.hostname;

    const [entryTld, ...entryDomainParts] = entryDomain.split(".").reverse();

    // Infer an app name from the base entry domain part 
    const defaultAppName = entryDomainParts[0].toLocaleLowerCase().replaceAll(
      /\w[a-z0-9]*[-_]*/g,
      match => {
        match = match.replace(/[-_]+/, " ");

        return match.charAt(0).toUpperCase() + match.slice(1).toLowerCase();
      }
    )
  
    // Infer an app ID from the entry domain by reversing it (e.g. `www.example.com` becomes `com.example.www`)
    const defaultAppId = entryTld + "." + entryDomainParts.join(".").replaceAll("-", "").toLocaleLowerCase();

    fs.writeFileSync(
      destinationFilepath.replace(/\.handlebars$/, ""),
      template({
        platform,
        entryDomain,
        appId: appId ?? defaultAppId,
        appName: appName ?? defaultAppName,
        entryUrl: navigationUrl || entryUrl,
        domainList: [entryDomain, ...additionalDomains].join("\\n"),
        smartDialerConfig: typeof smartDialerConfig === "object" ? JSON.stringify(smartDialerConfig).replaceAll('"',  '\\"') : smartDialerConfig,
        additionalDomains,
      }),
      "utf8",
    );
  }

  console.log("Copying mobileproxy files into the project...");

  fs.cpSync(
    SDK_MOBILEPROXY_OUTPUT_DIR,
    WRAPPER_APP_OUTPUT_SDK_MOBILEPROXY_DIR,
    { recursive: true },
  );

  console.log("Installing external dependencies for the project...");
  await promisify(exec)(`
    cd ${WRAPPER_APP_OUTPUT_DIR}
    npm install
    npx cap sync ${platform}
  `);

  console.log(`Zipping project to ${chalk.blue(WRAPPER_APP_OUTPUT_ZIP)}...`);
  await zip(WRAPPER_APP_OUTPUT_DIR, WRAPPER_APP_OUTPUT_ZIP);

  console.log("Project ready!");
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

  if (!args.platform) {
    throw new Error(`Parameter \`--platform\` not provided.`);
  }

  if (!args.entryDomain) {
    throw new Error(`Parameter \`--entryUrl\` not provided.`);
  }

  main({
    ...args,
    additionalDomains: args.additionalDomains?.split(/,\s*/) ?? [],
    smartDialerConfig: args.smartDialerConfig ? JSON.parse(args.smartDialerConfig) : undefined,
  })
    .catch(console.error);
}
