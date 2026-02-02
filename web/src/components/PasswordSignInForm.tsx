import { timestampDate } from "@bufbuild/protobuf/wkt";
import { AlertTriangle, LoaderIcon } from "lucide-react";
import { useState } from "react";
import { setAccessToken } from "@/auth-state";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { authServiceClient } from "@/connect";
import { useAuth } from "@/contexts/AuthContext";
import { useInstance } from "@/contexts/InstanceContext";
import useLoading from "@/hooks/useLoading";
import useNavigateTo from "@/hooks/useNavigateTo";
import { cn } from "@/lib/utils";
import { useTranslate } from "@/utils/i18n";

// Helper function to convert technical error messages to user-friendly ones
function getFriendlyErrorMessage(errorMessage: string, t: ReturnType<typeof useTranslate>): string {
  const lowerMessage = errorMessage.toLowerCase();

  // Network/connection errors
  if (lowerMessage.includes("network") || lowerMessage.includes("fetch") || lowerMessage.includes("connect")) {
    return t("auth.error.network");
  }
  // Authentication failures
  if (
    lowerMessage.includes("401") ||
    lowerMessage.includes("unauthorized") ||
    lowerMessage.includes("invalid") ||
    lowerMessage.includes("credentials")
  ) {
    return t("auth.error.invalid-credentials");
  }
  // User not found
  if (lowerMessage.includes("not found") || lowerMessage.includes("404")) {
    return t("auth.error.user-not-found");
  }
  // Server errors
  if (lowerMessage.includes("500") || lowerMessage.includes("502") || lowerMessage.includes("503")) {
    return t("auth.error.server");
  }
  // Rate limiting
  if (lowerMessage.includes("429") || lowerMessage.includes("rate limit") || lowerMessage.includes("too many")) {
    return t("auth.error.rate-limit");
  }
  // Default fallback
  return t("auth.error.default");
}

function PasswordSignInForm() {
  const t = useTranslate();
  const navigateTo = useNavigateTo();
  const { profile } = useInstance();
  const { initialize } = useAuth();
  const actionBtnLoadingState = useLoading(false);
  const [username, setUsername] = useState(profile.mode === "demo" ? "demo" : "");
  const [password, setPassword] = useState(profile.mode === "demo" ? "secret" : "");
  const [error, setError] = useState("");

  const handleUsernameInputChanged = (e: React.ChangeEvent<HTMLInputElement>) => {
    const text = e.target.value as string;
    setUsername(text);
    if (error) setError("");
  };

  const handlePasswordInputChanged = (e: React.ChangeEvent<HTMLInputElement>) => {
    const text = e.target.value as string;
    setPassword(text);
    if (error) setError("");
  };

  const handleFormSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    handleSignInButtonClick();
  };

  const handleSignInButtonClick = async () => {
    if (username === "" || password === "") {
      return;
    }

    if (actionBtnLoadingState.isLoading) {
      return;
    }

    try {
      actionBtnLoadingState.setLoading();
      const response = await authServiceClient.signIn({
        credentials: {
          case: "passwordCredentials",
          value: { username, password },
        },
      });
      // Store access token from login response
      if (response.accessToken) {
        setAccessToken(response.accessToken, response.accessTokenExpiresAt ? timestampDate(response.accessTokenExpiresAt) : undefined);
      }
      await initialize();
      navigateTo("/");
    } catch (error: unknown) {
      console.error(error);
      const errorMessage = error instanceof Error ? error.message : String(error);
      setError(getFriendlyErrorMessage(errorMessage, t));
    }
    actionBtnLoadingState.setFinish();
  };

  return (
    <form className="w-full mt-2" onSubmit={handleFormSubmit}>
      <div className="flex flex-col justify-start items-start w-full gap-4">
        {error && (
          <div className="w-full flex items-center gap-2 p-3 text-sm text-destructive bg-destructive/10 border border-destructive/20 rounded-md animate-in fade-in slide-in-from-top-1">
            <AlertTriangle className="w-4 h-4 shrink-0" />
            <span>{error}</span>
          </div>
        )}
        <div className="w-full flex flex-col justify-start items-start">
          <span className="leading-8 text-muted-foreground">{t("common.username")}</span>
          <Input
            type="text"
            readOnly={actionBtnLoadingState.isLoading}
            placeholder={t("common.username")}
            value={username}
            autoComplete="username"
            autoCapitalize="off"
            spellCheck={false}
            onChange={handleUsernameInputChanged}
            className={cn("w-full bg-background h-10", error && "border-destructive focus-visible:ring-destructive/50")}
            required
          />
        </div>
        <div className="w-full flex flex-col justify-start items-start">
          <span className="leading-8 text-muted-foreground">{t("common.password")}</span>
          <Input
            type="password"
            readOnly={actionBtnLoadingState.isLoading}
            placeholder={t("common.password")}
            value={password}
            autoComplete="current-password"
            autoCapitalize="off"
            spellCheck={false}
            onChange={handlePasswordInputChanged}
            className={cn("w-full bg-background h-10", error && "border-destructive focus-visible:ring-destructive/50")}
            required
          />
        </div>
      </div>
      <div className="flex flex-row justify-end items-center w-full mt-6">
        <Button type="submit" className="w-full h-10" disabled={actionBtnLoadingState.isLoading} onClick={handleSignInButtonClick}>
          {t("common.sign-in")}
          {actionBtnLoadingState.isLoading && <LoaderIcon className="w-5 h-auto ml-2 animate-spin opacity-60" />}
        </Button>
      </div>
    </form>
  );
}

export default PasswordSignInForm;
