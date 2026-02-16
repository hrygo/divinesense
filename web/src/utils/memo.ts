import type { Memo } from "@/types/proto/api/v1/memo_service_pb";
import { Visibility } from "@/types/proto/api/v1/memo_service_pb";

/**
 * Extract memo ID from memo.name
 *
 * @param memo - Memo object with name property (format: "memos/{id}")
 * @returns Extracted memo ID or full name if extraction fails
 */
export function getMemoId(memo: Memo): string {
  return memo.name.split("/").pop() || memo.name;
}

export const convertVisibilityFromString = (visibility: string) => {
  switch (visibility) {
    case "PUBLIC":
      return Visibility.PUBLIC;
    case "PROTECTED":
      return Visibility.PROTECTED;
    case "PRIVATE":
      return Visibility.PRIVATE;
    default:
      return Visibility.PUBLIC;
  }
};

export const convertVisibilityToString = (visibility: Visibility) => {
  switch (visibility) {
    case Visibility.PUBLIC:
      return "PUBLIC";
    case Visibility.PROTECTED:
      return "PROTECTED";
    case Visibility.PRIVATE:
      return "PRIVATE";
    default:
      return "PRIVATE";
  }
};
