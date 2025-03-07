import handlebars from "handlebars";
import fs from "fs";
import path from "path";
import { glob } from "glob";

if (!process.env.NGROK_DOMAIN) {
  throw new Error("NGROK_DOMAIN must be set!");
}

(async function () {
  let handlebarsTemplatePaths = [];

  try {
    handlebarsTemplatePaths = await glob(
      path.join(process.cwd(), "./**/*.handlebars"),
    );
  } catch (err) {
    throw new Error("Error finding .handlebars files");
  }

  for (const templatePath of handlebarsTemplatePaths) {
    try {
      const templateText = fs.readFileSync(templatePath, "utf-8");
      const template = handlebars.compile(templateText);

      fs.writeFileSync(
        templatePath.replace(".handlebars", ""),
        template({ NGROK_DOMAIN: process.env.NGROK_DOMAIN }),
        "utf-8",
      );

      console.log(`Compiled '${templatePath}'`);
    } catch (error) {
      console.error(`Error compiling ${templatePath}:`, error);
    }
  }
})();
