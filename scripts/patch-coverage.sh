#!/bin/bash
# Patch coverage reporter - shows coverage of lines added in this branch
# Similar to codecov's patch coverage feature

set -euo pipefail

MAIN_BRANCH="${1:-main}"
COVERAGE_FILE="coverage.out"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

if [ ! -f "$COVERAGE_FILE" ]; then
    echo "Error: $COVERAGE_FILE not found. Run 'make test-coverage' first."
    exit 1
fi

# Check if we're on a branch different from main
CURRENT_BRANCH=$(git branch --show-current)
if [ "$CURRENT_BRANCH" = "$MAIN_BRANCH" ]; then
    echo "Warning: Currently on $MAIN_BRANCH branch. Comparing with HEAD~1 instead."
    COMPARE_REF="HEAD~1"
else
    COMPARE_REF="$MAIN_BRANCH"
fi

echo "Analyzing patch coverage (comparing with $COMPARE_REF)..."
echo ""

# Create temporary files
ADDED_LINES_FILE=$(mktemp)
COVERAGE_DATA_FILE=$(mktemp)
RESULTS_FILE=$(mktemp)

cleanup() {
    rm -f "$ADDED_LINES_FILE" "$COVERAGE_DATA_FILE" "$RESULTS_FILE"
}
trap cleanup EXIT

# Get list of changed .go files and their added lines
# Format: filename:line_number
git diff "$COMPARE_REF...HEAD" --unified=0 --no-color -- '*.go' | \
    awk '
    /^\+\+\+ b\// {
        current_file = substr($0, 7)  # Remove "+++ b/" prefix
    }
    /^@@ / {
        # Parse hunk header: @@ -old_start,old_count +new_start,new_count @@
        match($0, /\+([0-9]+)(,([0-9]+))?/, arr)
        new_start = arr[1]
        new_count = arr[3] ? arr[3] : 1
        line_num = new_start
    }
    /^\+[^+]/ {
        # This is an added line (starts with + but not ++)
        if (current_file) {
            print current_file ":" line_num
            line_num++
        }
    }
    /^ / {
        # Context line, increment counter
        line_num++
    }
    ' > "$ADDED_LINES_FILE"

# Parse coverage.out to extract covered lines
# Format: filename:line_number:covered (1 or 0)
tail -n +2 "$COVERAGE_FILE" | \
    awk -F'[: ,]' '{
        file = $1
        # Strip module prefix (e.g., "bennypowers.dev/dtls/")
        sub(/^[^\/]+\/[^\/]+\//, "", file)
        # Line range is in format: line.col,line.col or just line.col
        split($2, start_parts, ".")
        line_start = start_parts[1]
        covered = $NF  # Last field is 0 or 1
        print file ":" line_start ":" covered
    }' > "$COVERAGE_DATA_FILE"

# Build associative arrays for analysis
declare -A file_added_lines
declare -A file_covered_lines
declare -A file_uncovered_lines

# Read added lines
while IFS=: read -r file line; do
    if [ -n "$file" ]; then
        file_added_lines["$file"]="${file_added_lines[$file]:-} $line"
    fi
done < "$ADDED_LINES_FILE"

# Check coverage for each added line
for file in "${!file_added_lines[@]}"; do
    for line in ${file_added_lines[$file]}; do
        # Check if this line has coverage data
        covered=$(grep "^$file:$line:" "$COVERAGE_DATA_FILE" | tail -1 | cut -d: -f3 || echo "")

        if [ "$covered" = "1" ]; then
            file_covered_lines["$file"]="${file_covered_lines[$file]:-} $line"
        elif [ "$covered" = "0" ]; then
            file_uncovered_lines["$file"]="${file_uncovered_lines[$file]:-} $line"
        fi
        # If no coverage data, assume it's not a statement (comments, braces, etc.)
    done
done

# Calculate statistics
total_added=0
total_covered=0
total_uncovered=0

echo "=== Patch Coverage by File ===" | tee "$RESULTS_FILE"
echo "" | tee -a "$RESULTS_FILE"

for file in $(printf '%s\n' "${!file_added_lines[@]}" | sort); do
    added_lines=(${file_added_lines[$file]})
    covered_lines=(${file_covered_lines[$file]:-})
    uncovered_lines=(${file_uncovered_lines[$file]:-})

    num_added=${#added_lines[@]}
    num_covered=${#covered_lines[@]}
    num_uncovered=${#uncovered_lines[@]}

    # Skip files with no testable lines
    if [ $((num_covered + num_uncovered)) -eq 0 ]; then
        continue
    fi

    total_added=$((total_added + num_covered + num_uncovered))
    total_covered=$((total_covered + num_covered))
    total_uncovered=$((total_uncovered + num_uncovered))

    patch_cov_pct=0
    if [ $((num_covered + num_uncovered)) -gt 0 ]; then
        patch_cov_pct=$((num_covered * 100 / (num_covered + num_uncovered)))
    fi

    # Color code based on coverage
    if [ $patch_cov_pct -ge 80 ]; then
        color=$GREEN
    elif [ $patch_cov_pct -ge 50 ]; then
        color=$YELLOW
    else
        color=$RED
    fi

    echo -e "${BLUE}$file${NC}" | tee -a "$RESULTS_FILE"
    echo -e "  Patch Coverage: ${color}${patch_cov_pct}%${NC} ($num_covered/$((num_covered + num_uncovered)) added lines covered)" | tee -a "$RESULTS_FILE"

    if [ $num_uncovered -gt 0 ]; then
        # Sort uncovered lines numerically
        uncovered_sorted=$(printf '%s\n' "${uncovered_lines[@]}" | sort -n | tr '\n' ' ' | sed 's/ $//')
        echo -e "  ${RED}Missing Lines:${NC} $uncovered_sorted" | tee -a "$RESULTS_FILE"
    fi
    echo "" | tee -a "$RESULTS_FILE"
done

# Calculate overall patch coverage
overall_patch_pct=0
if [ $total_added -gt 0 ]; then
    overall_patch_pct=$((total_covered * 10000 / total_added))
fi

# Format as percentage with 2 decimal places
overall_patch_pct_formatted=$(printf "%d.%02d" $((overall_patch_pct / 100)) $((overall_patch_pct % 100)))

echo "=== Summary ===" | tee -a "$RESULTS_FILE"
if [ $overall_patch_pct -ge 8000 ]; then
    color=$GREEN
elif [ $overall_patch_pct -ge 5000 ]; then
    color=$YELLOW
else
    color=$RED
fi
echo -e "${BLUE}Overall Patch Coverage:${NC} ${color}${overall_patch_pct_formatted}%${NC} ($total_covered/$total_added added lines)" | tee -a "$RESULTS_FILE"

# Calculate coverage change (if we can get baseline coverage)
if git rev-parse "$COMPARE_REF" >/dev/null 2>&1; then
    # Get total coverage from current coverage.out
    current_total=$(go tool cover -func="$COVERAGE_FILE" 2>/dev/null | tail -1 | awk '{print $NF}' | sed 's/%//' || echo "0")

    # Try to get coverage from main branch
    baseline_total="0"
    if git show "$COMPARE_REF:$COVERAGE_FILE" > /dev/null 2>&1; then
        baseline_total=$(git show "$COMPARE_REF:$COVERAGE_FILE" 2>/dev/null | go tool cover -func=/dev/stdin 2>/dev/null | tail -1 | awk '{print $NF}' | sed 's/%//' || echo "0")
    fi

    if [ "$baseline_total" != "0" ] && [ "$current_total" != "0" ]; then
        # Calculate change
        change=$(echo "$current_total - $baseline_total" | bc)

        # Format change with sign
        if (( $(echo "$change >= 0" | bc -l) )); then
            sign="+"
            color=$GREEN
        else
            sign=""
            color=$RED
        fi

        echo -e "${BLUE}Coverage Change:${NC} ${color}${sign}${change}%${NC} (${baseline_total}% → ${current_total}%)" | tee -a "$RESULTS_FILE"
    fi
fi

echo "" | tee -a "$RESULTS_FILE"

# Exit successfully (this is informational, not a gate)
exit 0
