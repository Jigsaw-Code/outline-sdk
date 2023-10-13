import { style } from "@vanilla-extract/css";
import { App } from "../../../../assets/themes";

export const Main = style({
  background: App.Theme.color.backgroundMuted,
  bottom: 0,
  padding: App.Theme.size.gap,
  position: "absolute",
  width: "100%",
  "@media": {
    '(prefers-contrast: more)': {
      borderTop: `${App.Theme.size.border} solid ${App.Theme.color.text}`,
      background: App.Theme.color.background,
    },
    [`only screen and (min-width: ${App.Theme.size.appWidthMin})`]: {
      display: "none"
    }
  }
});

export const Inner = style({
  alignItems: "center",
  display: "flex",
  gap: App.Theme.size.gap,
  margin: "0 auto",
  maxWidth: App.Theme.size.appWidthMax,
  padding: `0 ${App.Theme.size.gap}`,
});

export const Separator = style({
  color: App.Theme.color.textMuted,
  fontFamily: App.Theme.font.sansSerif,
  fontSize: App.Theme.size.fontSmall,
  "@media": {
    '(prefers-contrast: more)': {
      color: App.Theme.color.text,
    }
  }
});

const selectorInteraction = {
  background: App.Theme.color.backgroundHighlight,
  color: App.Theme.color.textHighlight,
};

export const Selector = style({
  background: App.Theme.color.textMuted,
  borderRadius: App.Theme.size.cornerRadius,
  color: App.Theme.color.backgroundMuted,
  cursor: "pointer",
  fontFamily: App.Theme.font.sansSerif,
  textAlignLast: "center",
  fontSize: App.Theme.size.fontSmall,
  padding: `${App.Theme.size.gapInner} ${App.Theme.size.gapNarrow}`,
  ':focus': selectorInteraction,
  ':hover': selectorInteraction,
  "@media": {
    '(prefers-contrast: more)': {
      background: App.Theme.color.text,
      color: App.Theme.color.background,
    }
  }
});
