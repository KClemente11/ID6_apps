#define _POSIX_C_SOURCE 200112L
#include <errno.h>
#include <stdio.h>
#include <string.h>
#include <sys/stat.h>
#include <unistd.h>

int main(int argc, char **argv)
{
    if (argc == 2 && strcmp(argv[1], "check") == 0)
        return 0;

    if (argc == 3 && strcmp(argv[1], "unlock") == 0) {
        if (rmdir(argv[2]) == 0)
            return 0;
        perror(argv[2]);
        return 1;
    }

    if (argc == 4 && strcmp(argv[1], "move") == 0) {
        struct stat destination;
        if (lstat(argv[3], &destination) == 0) {
            fprintf(stderr, "Destination already exists: %s\n", argv[3]);
            return 1;
        }
        if (errno != ENOENT) {
            perror(argv[3]);
            return 1;
        }
        if (rename(argv[2], argv[3]) == 0)
            return 0;
        fprintf(stderr, "Could not rename %s to %s: %s\n",
                argv[2], argv[3], strerror(errno));
        return 1;
    }

    fputs("Usage: gaugedeck-fileops check | unlock DIR | move SOURCE DEST\n", stderr);
    return 2;
}
