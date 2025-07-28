# listests

List Go tests in packages with their subtests.

## Usage

```bash
# Has ./... as default.
listests

# Specific packages
listests ./some/package ./another/package


# With build tags
listests -tags=integration ./some/package
```

## Example

```bash
$ listests
TestSimple
TestSubTests
TestSubTests/t1
TestSubTests/t2
TestNestedSubTests
TestNestedSubTests/t1
TestNestedSubTests/t1/t1
TestSubTestsWithGeneratedNames
TestSubTestsWithGeneratedNames/<fmt.Sprintf("t%v", i)>
TestTable
TestTable/t1
TestTable/t2
```

Use `--format` to customize the output, see shell functions below as an example.

## Misc

### Interactive with fzf + bat

Might be more simpler but works for me.

#### Fish
```fish
function gotaf
    argparse 'tags=' -- $argv
    or return

    set -l pkgs $argv
    if test -z "$pkgs"
        set pkgs ./...
    end

    set -l tags_opt
    if set -q _flag_tags
        set tags_opt "-tags=$_flag_tags"
    end

    set -l format "{{.RelativeDirectory}}:{{.FullDisplayName}}:{{.RelativeFileName}}:{{.Range.Start.Line}}:{{.Range.End.Line}}"
    set -l lines ( listests --format="$format" $tags_opt $pkgs | fzf --delimiter : \
        --multi \
        --preview 'echo $FZF_COLUMNS; bat --style=full --color=always --terminal-width $FZF_COLUMNS --highlight-line {4}:{5} {3}' \
        --preview-window '70%,~4,+{4}+4/4' \
        --height 60%
    )
    if test -z "$lines"
        return 0
    end

    set -l tests
    set -l packages
    for line in $lines
        set -l testname ( echo $line | cut -d : -f 2 )
        set -l pkg ( echo $line | cut -d : -f 1 )
        set tests $tests $testname
        set packages $packages "./$pkg"
    end

    set packages ( printf '%s\n' $packages | sort -u )

    set -l gotest_tags
    if set -q _flag_tags
        set gotest_tags -tags "$_flag_tags"
    end

    gotest -v $gotest_tags $packages -count=1 -run="$(string join '|' $tests)"
end
```

#### zsh

```zsh
gotaf() {
    local tags=""
    while [[ $# -gt 0 ]]; do
        case $1 in
            --tags=*)
                tags="${1#*=}"
                shift
                ;;
            *)
                break
                ;;
        esac
    done
    
    local pkgs=("$@")
    if [[ ${#pkgs[@]} -eq 0 ]]; then
        pkgs=("./...")
    fi
    
    local tags_opt=""
    if [[ -n "$tags" ]]; then
        tags_opt="-tags=$tags"
    fi
    
    local format="{{.RelativeDirectory}}:{{.FullDisplayName}}:{{.RelativeFileName}}:{{.Range.Start.Line}}:{{.Range.End.Line}}"
    local lines=($(listests --format="$format" $tags_opt "${pkgs[@]}" | fzf --delimiter : \
        --multi \
        --preview 'echo $FZF_COLUMNS; bat --style=full --color=always --terminal-width $FZF_COLUMNS --highlight-line {4}:{5} {3}' \
        --preview-window '70%,~4,+{4}+4/4' \
        --height 60%
    ))
    
    if [[ ${#lines[@]} -eq 0 ]]; then
        return 0
    fi
    
    local tests=()
    local packages=()
    for line in "${lines[@]}"; do
        local testname=$(echo "$line" | cut -d : -f 2)
        local pkg=$(echo "$line" | cut -d : -f 1)
        tests+=("$testname")
        packages+=("./$pkg")
    done
    
    packages=($(printf '%s\n' "${packages[@]}" | sort -u))
    
    local gotest_tags=()
    if [[ -n "$tags" ]]; then
        gotest_tags=(-tags "$tags")
    fi
    
    local tests_pattern=$(IFS='|'; echo "${tests[*]}")
    gotest -v "${gotest_tags[@]}" "${packages[@]}" -count=1 -run="$tests_pattern"
}
```
