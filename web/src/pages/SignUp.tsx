import { create } from "@bufbuild/protobuf";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import { LoaderIcon } from "lucide-react";
import { useState } from "react";
import { Link } from "react-router-dom";
import { setAccessToken } from "@/auth-state";
import { AuthErrorMessage } from "@/components/AuthErrorMessage";
import AuthFooter from "@/components/AuthFooter";
import { AuthSkeleton } from "@/components/AuthSkeleton";
import { ServiceUnavailable } from "@/components/ServiceUnavailable";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { authServiceClient, userServiceClient } from "@/connect";
import { useInstance } from "@/contexts/InstanceContext";
import useLoading from "@/hooks/useLoading";
import { User_Role, UserSchema } from "@/types/proto/api/v1/user_service_pb";
import { useTranslate } from "@/utils/i18n";

// Helper function to convert technical error messages to user-friendly ones
function getFriendlyErrorMessage(errorMessage: string, username: string, t: ReturnType<typeof useTranslate>): string {
  const lowerMessage = errorMessage.toLowerCase();

  // Invalid username format - detect if user tried to use email
  if (lowerMessage.includes("invalid username")) {
    if (username.includes("@")) {
      return t("auth.error.username-email-format");
    }
    return t("auth.error.username-format");
  }
  // Username already exists
  if (lowerMessage.includes("already exists") || lowerMessage.includes("duplicate") || lowerMessage.includes("conflict")) {
    return t("auth.error.username-exists");
  }
  // Network/connection errors
  if (lowerMessage.includes("network") || lowerMessage.includes("fetch") || lowerMessage.includes("connect")) {
    return t("auth.error.network");
  }
  // Other invalid input errors
  if (lowerMessage.includes("invalid") || lowerMessage.includes("validation") || lowerMessage.includes("format")) {
    return t("auth.error.invalid-input");
  }
  // Server errors
  if (lowerMessage.includes("500") || lowerMessage.includes("502") || lowerMessage.includes("503")) {
    return t("auth.error.server");
  }
  // Rate limiting
  if (lowerMessage.includes("429") || lowerMessage.includes("rate limit") || lowerMessage.includes("too many")) {
    return t("auth.error.rate-limit", { count: "5" });
  }
  // Default fallback
  return t("auth.error.sign-up-failed");
}

const SignUp = () => {
  const t = useTranslate();
  const actionBtnLoadingState = useLoading(false);
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const { generalSetting: instanceGeneralSetting, profile, isLoading: instanceLoading, isServiceAvailable } = useInstance();

  // Show loading state while instance config is loading
  if (instanceLoading) {
    return <AuthSkeleton />;
  }

  // Show service unavailable message if backend is not reachable
  if (!isServiceAvailable) {
    return <ServiceUnavailable showDetails fullscreen={false} />;
  }

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
    handleSignUpButtonClick();
  };

  const handleSignUpButtonClick = async () => {
    if (username.trim() === "" || password.trim() === "") {
      setError(t("auth.error.empty-fields"));
      return;
    }

    if (actionBtnLoadingState.isLoading) {
      return;
    }

    try {
      actionBtnLoadingState.setLoading();
      const user = create(UserSchema, {
        username: username.trim(),
        password,
        role: User_Role.USER,
      });
      await userServiceClient.createUser({ user });
      const response = await authServiceClient.signIn({
        credentials: {
          case: "passwordCredentials",
          value: { username: username.trim(), password },
        },
      });
      // Store access token from login response
      if (response.accessToken) {
        setAccessToken(response.accessToken, response.accessTokenExpiresAt ? timestampDate(response.accessTokenExpiresAt) : undefined);
      }
      window.location.href = "/";
    } catch (error: unknown) {
      console.error(error);
      const errorMessage = error instanceof Error ? error.message : String(error);
      setError(getFriendlyErrorMessage(errorMessage, username.trim(), t));
    }
    actionBtnLoadingState.setFinish();
  };

  return (
    <div className="py-4 sm:py-8 w-80 max-w-full min-h-svh mx-auto flex flex-col justify-start items-center">
      <div className="w-full py-4 grow flex flex-col justify-center items-center">
        <div className="w-full flex flex-row justify-center items-center mb-6">
          <img className="h-14 w-auto rounded-full shadow" src={instanceGeneralSetting.customProfile?.logoUrl || "/logo.webp"} alt="" />
          <p className="ml-2 text-5xl text-foreground opacity-80">{instanceGeneralSetting.customProfile?.title || t("app.name")}</p>
        </div>

        {/* Registration disabled message */}
        {instanceGeneralSetting.disallowUserRegistration ? (
          <p className="w-full text-center text-lg text-muted-foreground px-4 py-6 bg-muted/30 rounded-lg">
            {t("auth.sign-up-not-allowed")}
          </p>
        ) : (
          <>
            <p className="w-full text-2xl text-muted-foreground">{t("auth.create-your-account")}</p>
            <form className="w-full mt-4" onSubmit={handleFormSubmit}>
              {/* Error message */}
              {error && <AuthErrorMessage message={error} />}

              <div className="flex flex-col justify-start items-start w-full gap-4">
                <div className="w-full flex flex-col justify-start items-start">
                  <label htmlFor="username" className="leading-8 text-muted-foreground text-sm font-medium">
                    {t("common.username")}
                  </label>
                  <Input
                    id="username"
                    className="w-full bg-background h-11"
                    type="text"
                    readOnly={actionBtnLoadingState.isLoading}
                    placeholder={t("common.username")}
                    value={username}
                    autoComplete="username"
                    autoCapitalize="off"
                    spellCheck={false}
                    onChange={handleUsernameInputChanged}
                    required
                  />
                </div>
                <div className="w-full flex flex-col justify-start items-start">
                  <label htmlFor="password" className="leading-8 text-muted-foreground text-sm font-medium">
                    {t("common.password")}
                  </label>
                  <Input
                    id="password"
                    className="w-full bg-background h-11"
                    type="password"
                    readOnly={actionBtnLoadingState.isLoading}
                    placeholder={t("common.password")}
                    value={password}
                    autoComplete="new-password"
                    autoCapitalize="off"
                    spellCheck={false}
                    onChange={handlePasswordInputChanged}
                    required
                  />
                </div>
              </div>
              <div className="flex flex-row justify-end items-center w-full mt-6">
                <Button type="submit" className="w-full h-11" disabled={actionBtnLoadingState.isLoading} onClick={handleSignUpButtonClick}>
                  {actionBtnLoadingState.isLoading ? (
                    <>
                      <LoaderIcon className="w-5 h-5 animate-spin opacity-60" />
                      <span className="ml-2">{t("common.loading")}</span>
                    </>
                  ) : (
                    t("common.sign-up")
                  )}
                </Button>
              </div>
            </form>
          </>
        )}

        {/* Sign-in tip - always reserve space to prevent layout shift */}
        <div className="h-7 mt-4">
          {!profile.owner ? (
            <p className="w-full text-sm text-center text-muted-foreground">{t("auth.host-tip")}</p>
          ) : (
            <p className="w-full text-sm text-center">
              <span className="text-muted-foreground">{t("auth.sign-in-tip")}</span>
              <Link to="/auth" className="cursor-pointer ml-1 text-primary hover:underline font-medium" viewTransition>
                {t("common.sign-in")}
              </Link>
            </p>
          )}
        </div>
      </div>
      <AuthFooter />
    </div>
  );
};

export default SignUp;
