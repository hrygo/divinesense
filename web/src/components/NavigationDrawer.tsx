import { useEffect, useState } from "react";
import { useLocation } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { Sheet, SheetContent, SheetTitle, SheetTrigger } from "@/components/ui/sheet";
import { useInstance } from "@/contexts/InstanceContext";
import Navigation from "./Navigation";

const NavigationDrawer = () => {
  const location = useLocation();
  const [open, setOpen] = useState(false);
  const { generalSetting } = useInstance();
  const title = generalSetting.customProfile?.title || "Memos";

  useEffect(() => {
    setOpen(false);
  }, [location.key]);

  return (
    <Sheet open={open} onOpenChange={setOpen}>
      <SheetTrigger asChild>
        <Button variant="ghost" className="min-h-[44px] min-w-[44px] px-0 hover:bg-transparent cursor-pointer">
          <img src="/logo.webp" alt={title} className="h-10 w-auto object-contain logo-animated" />
        </Button>
      </SheetTrigger>
      <SheetContent side="left" className="w-72 max-w-full overflow-auto px-4 pt-2 bg-background [&_.absolute.top-4.right-4]:hidden">
        <SheetTitle className="sr-only">Navigation</SheetTitle>
        <Navigation className="pb-4" />
      </SheetContent>
    </Sheet>
  );
};

export default NavigationDrawer;
