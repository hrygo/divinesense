import i18n, { BackendModule, FallbackLng, FallbackLngObjList } from "i18next";

import { initReactI18next } from "react-i18next";
import { findNearestMatchedLanguage } from "./utils/i18n";

export const locales = ["en", "zh-Hans"];

const fallbacks = {
  zh: ["zh-Hans", "en"],
} as FallbackLngObjList;

const LazyImportPlugin: BackendModule = {
  type: "backend",
  init: function () {},
  read: function (language, _, callback) {
    const matchedLanguage = findNearestMatchedLanguage(language);
    import(`./locales/${matchedLanguage}.json`)
      .then((translation: Record<string, unknown>) => {
        callback(null, translation);
      })
      .catch(() => {
        // Fallback to English.
      });
  },
};

i18n
  .use(LazyImportPlugin)
  .use(initReactI18next)
  .init({
    detection: {
      order: ["navigator"],
    },
    fallbackLng: {
      ...fallbacks,
      ...{ default: ["en"] },
    } as FallbackLng,
  });

export default i18n;
export type TLocale = (typeof locales)[number];
