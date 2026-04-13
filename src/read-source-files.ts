import { join } from "@std/path";

/**
 * Regular expression to match a vehicle (trailer or truck) with or without company at the end
 */
const vehicleRegex =
  /^traffic_(vehicle|trailer)\s+:\s+traffic\.(\S+)(\s+\/\/\s+(\S+))?/;

/**
 * A vehicle type
 */
export type VehicleType = "trailer" | "truck";

/**
 * List of vehicle types
 */
export const vehicleTypes: VehicleType[] = ["trailer", "truck"];

/**
 * A vehicle
 */
export interface Vehicle {
  /**
   * File name of vehicle
   */
  fileName: string;

  /**
   * Name of vehicle
   */
  name: string;
}

/**
 * A company
 */
export interface Company {
  /**
   * List of trailers
   */
  trailers: Vehicle[];

  /**
   * List of trucks
   */
  trucks: Vehicle[];
}

/**
 * Map of companies and their vehicles
 */
export interface CompanyMap {
  [company: string]: Company;
}

/**
 * Recursively read sui-files
 *
 * @param directory Directory to start recursion on
 * @param companyMap Company map to fill with info
 */
export async function readSourceFiles(
  directory: string,
  companyMap: CompanyMap,
) {
  console.info(`[INFO] Iterating over all files in '${directory}...`);

  // iterate over all entries in directory
  for await (const directoryEntry of Deno.readDir(directory)) {
    const directoryEntryPath = join(directory, directoryEntry.name);

    const fileInfo = await Deno.lstat(directoryEntryPath);

    if (fileInfo.isDirectory) {
      await readSourceFiles(directoryEntryPath, companyMap);

      continue;
    }

    // check if file has correct extension
    if (
      !fileInfo.isFile ||
      (
        !directoryEntry.name.endsWith(".sui") &&
        !directoryEntry.name.endsWith(".sii")
      )
    ) {
      console.info(
        `[INFO] Ignoring '${directoryEntryPath}' because it's not a file/has wrong extension...`,
      );

      continue;
    }

    console.info(`[INFO] Reading text file '${directoryEntryPath}'...`);

    // read text file
    const content = await Deno.readTextFile(directoryEntryPath);

    // split content into lines
    const lines = content.split("\n");

    // iterate over all lines
    for (const line of lines) {
      // try to match/find data
      const match = line.match(vehicleRegex);

      // check if data was found
      if (!Array.isArray(match)) {
        console.warn(
          `[WARN] Could not extract relevant data from '${directoryEntryPath}'. Skipping it...`,
        );

        continue;
      }

      // get type from match
      const type: VehicleType = match[1] === "vehicle" ? "truck" : "trailer";

      // get name from matches
      const name = match[2];

      // get company from matches
      const company = match[4]?.toLowerCase().replace(/\//, "_").trim() ??
        "plain";

      // check if company is already in map
      if (!(company in companyMap)) {
        console.info(`[INFO] Adding new company '${company}'.`);

        // initialize list for company
        companyMap[company] = {
          trailers: [],
          trucks: [],
        };
      }

      console.info(
        `[INFO] Adding ${type} '${name}' to company '${company}'...`,
      );

      // add vehicle to company list
      companyMap[company][`${type}s`].push({
        name,
        fileName: directoryEntryPath,
      });
    }
  }
}
