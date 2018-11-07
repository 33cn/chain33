#!/usr/bin/env bash
guodun1="$(../chain33-cli config config_tx -k token-finisher -o del -v 13KDST7ndBmzunGYAkCRabooWGRc95hM3W --paraName "user.p.starsunny.")"
guodun2="$(../chain33-cli config config_tx -k token-finisher -o add -v 1AtQcjpDYaJSxvAE3aacfJUknsFNb7eWLd --paraName "user.p.starsunny.")"
echo "cli wallet sign -d $guodun1 -a 1JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP -e 1h > d:/b.txt"
echo "cli wallet sign -d $guodun2 -a 1JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP -e 1h >> d:/b.txt"
