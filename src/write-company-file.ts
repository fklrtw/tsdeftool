import { join } from "@std/path";

/**
 * Write company file
 *
 * @param companyName Name of company to write file for
 * @param type Type of vehicle
 * @param vehicles List of vehicles
 * @param destinationDirectory Directory to write file to
 */
export async function writeCompanyFile(
  companyName: string,
  type: "trailer" | "truck",
  vehicles: string[],
  destinationDirectory: string,
) {
  // set filename for company
  const fileName = `traffic.${companyName}${
    companyName !== "plain" ? `_${type}` : ""
  }.sii`;

  console.info(
    `[OK] Generating new file '${fileName}' for company '${companyName}' with ${vehicles.length} ${type}(s)'...`,
  );

  let contents = `SiiNunit
{\n`;

  for (const vehicle of vehicles) {
    contents += `\ncountry_traffic_info : .country.info.traffic.${vehicle} {
    object: traffic.${vehicle}
    spawn_frequency : 0.00
}\n`;
  }

  contents += "}\n";

  await Deno.writeTextFile(
    join(destinationDirectory, fileName),
    contents,
  );
}
