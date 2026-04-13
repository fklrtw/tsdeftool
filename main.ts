import { Command } from "@cliffy/command";
import { HelpCommand } from "@cliffy/command/help";
import { CompletionsCommand } from "@cliffy/command/completions";
import { assertDirectory } from "./src/assert-directory.ts";
import {
  type CompanyMap,
  readSourceFiles,
  vehicleTypes,
} from "./src/read-source-files.ts";
import { writeCompanyFile } from "./src/write-company-file.ts";
import { join } from "@std/path";
import { levenshteinDistance } from "@std/text";
import denoConfig from "./deno.json" with { type: "json" };

await new Command()
  .name(denoConfig.name)
  .description("Adjust files for American Truck Simulator")
  .version(denoConfig.version)
  .option(
    "--source-directory <value>",
    "Source directory/where the source files are",
    {
      default: Deno.cwd(),
    },
  )
  .option(
    "--destination-directory <value>",
    "Destination directory/where the generated files should go",
    {
      default: join(Deno.cwd(), "output"),
    },
  )
  .option(
    "--maximum-levenshtein-distance <value>",
    "(Maximum) Levenshtein distance of two company names to be considered a typo",
    {
      default: "2",
      value: (maximumLevenshteinDistance) => {
        return parseInt(maximumLevenshteinDistance);
      },
    },
  )
  .action(
    async (
      {
        sourceDirectory,
        destinationDirectory,
        maximumLevenshteinDistance,
      },
    ) => {
      await assertDirectory(sourceDirectory, "source", true);
      await assertDirectory(destinationDirectory, "destination", true);

      console.time("ok");

      // map of companies with a list of their trucks
      const companyMap: CompanyMap = {};

      await readSourceFiles(sourceDirectory, companyMap);

      console.info(
        `\n[INFO] Writing ${
          Object.keys(companyMap).length
        } new file(s) to '${destinationDirectory}'...`,
      );

      const companyNames = Object.keys(companyMap).sort();

      const glossary: [string, string, string, string][] = [
        ["company", "type", "vehicle", "file"],
      ];

      // iterate over all companies in map
      for (const companyName of companyNames) {
        for (const type of vehicleTypes) {
          if (companyMap[companyName][`${type}s`].length > 0) {
            await writeCompanyFile(
              companyName,
              type,
              companyMap[companyName][`${type}s`].map((entry) => entry.name),
              destinationDirectory,
            );

            companyMap[companyName][`${type}s`]
              .sort((entryA, entryB) => entryA.name.localeCompare(entryB.name))
              .forEach((entry) =>
                glossary.push([companyName, type, entry.name, entry.fileName])
              );
          }
        }
      }

      const companyTypos: [string, string, string][] = [
        ["company A", "company B", "distance"],
      ];

      for (let i = 0; i < companyNames.length; i++) {
        const companyA = companyNames[i];

        for (const companyB of companyNames.slice(i + 1)) {
          const distance = levenshteinDistance(companyA, companyB);

          if (distance <= maximumLevenshteinDistance) {
            companyTypos.push([
              companyA,
              companyB,
              distance.toFixed(0),
            ]);
          }
        }
      }

      await Deno.writeTextFile(
        join(destinationDirectory, "_typos.csv"),
        companyTypos.map((row) => row.join(";")).join("\n"),
      );

      await Deno.writeTextFile(
        join(destinationDirectory, "_glossary.csv"),
        glossary.map((row) => row.join(";")).join("\n"),
      );

      console.timeEnd("ok");
    },
  )
  .command("help", new HelpCommand())
  .command("completions", new CompletionsCommand())
  .parse(Deno.args);
