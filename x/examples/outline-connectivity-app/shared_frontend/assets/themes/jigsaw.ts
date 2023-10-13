import { createTheme } from '@vanilla-extract/css';

export const [Class, Theme] = createTheme({
  color: {
    black: 'hsl(300, 3%, 7%)',
    white: 'white',
    green: 'hsl(153, 39%, 15%)',
    greenMedium: 'hsl(156, 33%, 31%)',
    greenLight: 'hsl(159, 13%, 74%)',
    gray: 'hsl(0, 0%, 73%)',
    grayMedium: 'hsl(0, 0%, 90%)',
    grayLight: 'hsl(0, 0%, 98%)',
    red: 'hsl(0, 100%, 50%)',
  },
});
