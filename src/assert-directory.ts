/**
 * Make sure, that a directory exists
 * @param directory Directory to check
 * @param identifier Identifier for directory
 * @param create Whether to create the directory, if it does not exist
 */
export async function assertDirectory(
  directory: string,
  identifier: string,
  create: boolean,
) {
  try {
    // get info for path
    const sourceDirectoryInfo = await Deno.lstat(directory);

    // check if path is a directory
    if (!sourceDirectoryInfo.isDirectory) {
      throw new Error(
        `[ERROR] Provided ${identifier} directory '${directory}' is not a directory.`,
      );
    }
  } catch (error) {
    if (
      typeof error === "object" && error !== null && "code" in error &&
      error.code === "ENOENT" && create
    ) {
      console.info(`[INFO] Creating ${identifier} directory '${directory}'.`);
      await Deno.mkdir(directory, { recursive: true });
    } else {
      throw new Error(
        `[ERROR] Provided ${identifier} directory '${directory}' does not exist.`,
      );
    }
  }
}
