#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
dist_dir="$root_dir/.jknobman133"
zip_path="$root_dir/JKnobMan133-jar.zip"
helper_src="$root_dir/tools/JKnobManRenderCli.java"
helper_classes="$dist_dir/helper-classes"
jar_path="$dist_dir/JKnobMan.jar"
class_path="$helper_classes:$jar_path"

if [[ ! -f "$jar_path" ]]; then
    mkdir -p "$dist_dir"
    unzip -oq "$zip_path" -d "$dist_dir"
fi

if [[ ! -f "$helper_classes/JKnobManRenderCli.class" || "$helper_src" -nt "$helper_classes/JKnobManRenderCli.class" ]]; then
    mkdir -p "$helper_classes"
    javac -cp "$jar_path" -d "$helper_classes" "$helper_src"
fi

exec java -Djava.awt.headless=true -Djava.application.path="$dist_dir" -cp "$class_path" JKnobManRenderCli "$@"
