import { Outlet } from "react-router-dom";
import NavigationDrawer from "@/components/NavigationDrawer";
import RouteHeaderImage from "@/components/RouteHeaderImage";
import useMediaQuery from "@/hooks/useMediaQuery";
import { cn } from "@/lib/utils";

/**
 * GeneralLayout - Layout for pages without sidebar
 *
 * Used by: Inboxes, Attachments, KnowledgeGraph
 *
 * Spacing specification (unified with MemoLayout):
 * - Mobile top padding: pt-4 (16px)
 * - Desktop top padding: pt-6 (24px)
 * - Bottom padding: pb-8 (32px)
 * - Horizontal padding: px-4 / sm:px-6
 * - Max width: max-w-[100rem] (1600px)
 *
 * @see docs/research/layout-spacing-unification.md
 */
const GeneralLayout = () => {
  const sm = useMediaQuery("sm");

  return (
    <section className="w-full h-full flex flex-col justify-start items-center overflow-hidden">
      {/* Mobile Header - Fixed height h-14 with py-2 for vertical spacing */}
      {!sm && (
        <div className="w-full flex items-center justify-center px-4 py-2 h-14 shrink-0 border-b border-border/50 bg-background sticky top-0 z-10 overflow-hidden relative">
          <div className="absolute left-4 top-0 bottom-0 flex items-center">
            <NavigationDrawer />
          </div>
          <RouteHeaderImage />
        </div>
      )}

      {/* Main Content - Unified spacing with other layouts */}
      <div className="w-full h-full overflow-y-auto">
        <div className={cn("w-full mx-auto px-4 sm:px-6 pt-4 sm:pt-6 pb-8", "max-w-[100rem]")}>
          <Outlet />
        </div>
      </div>
    </section>
  );
};

export default GeneralLayout;
