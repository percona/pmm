#! env perl
# Builds a sed command file for replacing replacements (from .res/replace.txt)
# with their text equivalents.
while (<STDIN>) {
    chomp;
    my @r = split ("replace::");
    $r[0] =~ s/^.. //;
    $r[0] =~ s/^\s+//;
    $r[0] =~ s/\s+$//;
    $r[1] =~ s/^\s+//;
    $r[1] =~ s/\s+$//;
    print "s~$r[0]~$r[1]~g\n";
}