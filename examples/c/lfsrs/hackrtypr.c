#include <stdio.h>
#include <stdlib.h>
#include <memory.h>
#include <unistd.h>
#include <time.h>
#include <fcntl.h>

#include <termios.h>

static struct termios orig_termios;  /* TERMinal I/O Structure */

static int ttyfd = STDIN_FILENO;     /* STDIN_FILENO is 0 by default */


void fatal(char *message)
{
  fprintf(stderr,"fatal error: %s\n",message);
  exit(1);
}

void tty_raw(void)
{
  struct termios raw;

  raw = orig_termios;  /* copy original and then modify below */

  /* input modes - clear indicated ones giving: no break, no CR to NL, 
     no parity check, no strip char, no start/stop output (sic) control */
  raw.c_iflag &= ~(BRKINT | ICRNL | INPCK | ISTRIP | IXON);

  /* output modes - clear giving: no post processing such as NL to CR+NL */
  raw.c_oflag &= ~(OPOST);

  /* control modes - set 8 bit chars */
  raw.c_cflag |= (CS8);

  /* local modes - clear giving: echoing off, canonical off (no erase with 
     backspace, ^U,...),  no extended functions, no signal chars (^Z,^C) */
  raw.c_lflag &= ~(ECHO | ICANON | IEXTEN | ISIG);

  /* control chars - set return condition: min number of bytes and timer */
  raw.c_cc[VMIN] = 5; raw.c_cc[VTIME] = 8; /* after 5 bytes or .8 seconds
                                              after first byte seen      */
  raw.c_cc[VMIN] = 0; raw.c_cc[VTIME] = 0; /* immediate - anything       */
  raw.c_cc[VMIN] = 2; raw.c_cc[VTIME] = 0; /* after two bytes, no timer  */
  raw.c_cc[VMIN] = 0; raw.c_cc[VTIME] = 8; /* after a byte or .8 seconds */

  /* put terminal in raw mode after flushing */
  if (tcsetattr(ttyfd,TCSAFLUSH,&raw) < 0) fatal("can't set raw mode");
}

int tty_reset(void)
{
  /* flush and reset */
  if (tcsetattr(ttyfd,TCSAFLUSH,&orig_termios) < 0) return -1;
  return 0;
}

void tty_atexit(void)  /* NOTE: If the program terminates due to a signal   */
{                      /* this code will not run.  This is for exit()'s     */
  tty_reset();        /* only.  For resetting the terminal after a signal, */
}                      /* a signal handler which calls tty_reset is needed. */


int main ( int argc, char** argv ) {
  FILE *in;
  int jj, nchars, res, flags, ch;
  // input file: take from argv[0]
  // rand()

  if (tcgetattr(ttyfd,&orig_termios) < 0) fatal("can't get tty settings");
  if (atexit(tty_atexit) != 0) fatal("atexit: can't register tty reset");
  tty_raw();

  if ( argc < 2 ) {
    printf("sorry, you're not l33t enough\r\n");
    exit(1);
  }

  in = fopen(argv[1],"r");

  flags = fcntl(0,F_GETFL,0);
  flags |= O_NONBLOCK;
  res = fcntl(0,F_SETFL,flags);

  for (;;) {
    // if they typed anything, 
    static char input[16] = {0};
    memset(input,'\0',sizeof(input)*sizeof(char));
    res = read(0, &input, sizeof(input)*sizeof(char));
    nchars = 1 + 2 * ((double) rand() / (double)RAND_MAX);
    // printf("nchars:%d\r\n",nchars);
    if ( res < 1 ) {
      usleep(100);
      continue;
    }
    // printf("res:%d\n",res);

    if (3 == input[0] ) // CTRL-C
      return 0;

    for (jj = 0; jj < nchars; ++jj) {
      ch = getc(in);
      if (-1 == ch) { return 0; }
      //if ( '\n' == ch ) {
        //putc('\r',stdout);
      //}
      putc(ch,stdout);
      fflush(stdout);
    }
  }
  return 0;
}
