import { Outlet } from "react-router-dom";
import NavigationDrawer from "@/components/NavigationDrawer";
import RouteHeaderImage from "@/components/RouteHeaderImage";
import useMediaQuery from "@/hooks/useMediaQuery";
import { cn } from "@/lib/utils";

/**
 * GeneralLayout - Layout for pages without sidebar (full-width like AIChat)
 *
 * Used by: Inboxes, Attachments, KnowledgeGraph, Review, Setting, MemoDetail
 *
 * Design specification:
 * - Full-width layout (same as AIChat for consistency)
 * - Mobile top padding: pt-4 (16px)
 * - Desktop top padding: pt-6 (24px)
 * - Bottom padding: pb-8 (32px)
 * - Horizontal padding: px-4 / sm:px-6
 * - No max-width limit (full-width app experience)
 *
 * @see docs/research/layout-spacing-unification.md
 */
const GeneralLayout = () => {
  const sm = useMediaQuery("sm");

  return (
    <section className="w-full h-full flex flex-col overflow-hidden">
      {/* Mobile Header - Fixed height h-14 with py-2 for vertical spacing */}
      {!sm && (
        <div className="w-full flex items-center justify-center px-4 py-2 h-14 shrink-0 border-b border-border/50 bg-background sticky top-0 z-10 overflow-hidden relative">
          <div className="absolute left-4 top-0 bottom-0 flex items-center">
            <NavigationDrawer />
          </div>
          <RouteHeaderImage />
        </div>
      )}

      {/* Main Content - Full width like AIChat */}
      <div className="w-full h-full overflow-y-auto">
        <div className={cn("w-full px-4 sm:px-6 pt-4 sm:pt-6 pb-8")}>
          <Outlet />
        </div>
      </div>
    </section>
  );
};

export default GeneralLayout;
