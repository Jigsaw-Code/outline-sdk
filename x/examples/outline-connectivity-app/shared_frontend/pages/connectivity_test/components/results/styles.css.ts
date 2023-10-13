import { style } from "@vanilla-extract/css";
import { App } from "../../../../assets/themes";

export const Header = style({
  display: "flex",
  justifyContent: "space-between",
  alignItems: "center",
  padding: `${App.Theme.size.gapNarrow} ${App.Theme.size.gap}`,
  "@media": {
    "(prefers-contrast: more)": {
      border: `${App.Theme.size.border} solid ${App.Theme.color.text}`,
      background: App.Theme.color.background
    }
  }
});

export const HeaderText = style({
  color: App.Theme.color.text,
  fontFamily: App.Theme.font.sansSerif,
  fontSize: App.Theme.size.font,
});

const opacityReset = {
  opacity: 1
}

export const HeaderClose = style({
  color: App.Theme.color.text,
  cursor: "pointer",
  fontFamily: App.Theme.font.sansSerif,
  fontSize: App.Theme.size.font,
  opacity: 0.5,
  ':hover': opacityReset,
  '@media': {
    '(prefers-contrast: more)': opacityReset
  }
});
