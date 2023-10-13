import { style } from "@vanilla-extract/css";
import { App } from "../../../../../assets/themes";

export const Main = style({
  listStyle: "none",
});

const highContrastItem = {
  background: App.Theme.color.background,
  borderWidth: `0 ${App.Theme.size.border} ${App.Theme.size.border} ${App.Theme.size.border}`,
  borderBottom: `${App.Theme.size.border} solid ${App.Theme.color.text}`,
}

const item = {
  alignItems: "center",
  display: "flex",
  gap: App.Theme.size.gap,
  padding: App.Theme.size.gap,
  borderWidth: `0 0 ${App.Theme.size.border} 0`,
  ':last-child': {
    borderBottomLeftRadius: App.Theme.size.cornerRadius,
    borderBottomRightRadius: App.Theme.size.cornerRadius,
    borderBottom: "none",
  },
  '@media': {
    '(prefers-contrast: more)': {
      ...highContrastItem,
      ':last-child': highContrastItem
    }
  }
};

export const SuccessItem = style({
  ...item,
  background: App.Theme.color.successBackground,
  borderColor: App.Theme.color.successText,
  selectors: {
    '& *': {
      color: App.Theme.color.successText,
    },
  },
});

export const FailureItem = style({
  ...item,
  background: App.Theme.color.errorBackground,
  borderColor: App.Theme.color.errorText,
  selectors: {
    '& *': {
      color: App.Theme.color.errorText,
    },
  },
});

export const ItemStatus = style({
  fontFamily: App.Theme.font.sansSerif,
  fontSize: App.Theme.size.font,
  flexShrink: 0,
});

export const ItemData = style({
  display: "grid",
  flexGrow: 1,
  gridAutoFlow: "column",
  gridAutoColumns: "1fr",
  gridTemplateRows: "repeat(2, min-content)",
  rowGap: App.Theme.size.gapNarrow,
  columnGap: App.Theme.size.gapNarrow,
  "@media": {
    [`only screen and (min-width: ${App.Theme.size.appWidthMin})`]: {
      display: "flex",
      flexDirection: "column",
      gap: App.Theme.size.gapInner
    }
  }
});

const highContrastItemDataMedia = {
  '@media': {
    '(prefers-contrast: more)': {
      opacity: 1,
      color: App.Theme.color.text,
    }
  }
}

export const ItemDataKey = style({
  opacity: 0.5,
  fontFamily: App.Theme.font.sansSerif,
  fontSize: App.Theme.size.fontSmall,
  textTransform: "uppercase",
  gridRowStart: 1,
  ...highContrastItemDataMedia
});

export const ItemDataValue = style({
  fontFamily: App.Theme.font.monospace,
  fontSize: App.Theme.size.font,
  gridRowStart: 2,
  ...highContrastItemDataMedia
});
