import pandas as pd
import json
import sys

def main():
    if len(sys.argv) < 3:
        print("Usage: python generate_slow_queries_table.py <input_csv_file> <output_md_file>")
        sys.exit(1)

    input_file = sys.argv[1]
    output_file = sys.argv[2]

    try:
        df = pd.read_csv(input_file)
    except FileNotFoundError:
        print(f"Error: File not found at {input_file}")
        sys.exit(1)

    # --- 1. Find slow domains ---
    median_durations = df.groupby(['domain', 'query_type'])['duration_ms'].median().unstack()
    
    if 'A' not in median_durations.columns or 'HTTPS' not in median_durations.columns:
        print("Data for 'A' or 'HTTPS' queries not found.")
        return

    median_durations['duration_diff'] = median_durations['HTTPS'] - median_durations['A']
    slow_domains_series = median_durations[median_durations['duration_diff'] > 50].index

    if slow_domains_series.empty:
        print("No domains found where median HTTPS query is > 50ms slower than median A query.")
        return

    # --- 2. Get all runs for slow domains ---
    slow_df = df[df['domain'].isin(slow_domains_series)]

    # --- 3. Calculate fastest HTTPS - fastest A ---
    min_durations = slow_df.groupby(['domain', 'query_type'])['duration_ms'].min().unstack()
    min_durations['min_diff'] = min_durations['HTTPS'] - min_durations['A']
    
    # Sort domains by the new difference column
    sorted_slow_domains = min_durations.sort_values(by='min_diff', ascending=False).index

    # --- 4. Generate Markdown Table ---
    header_markdown = "| No. | Domain | Rank | Min HTTPS - Min A | Run 1 | Run 2 | Run 3 | Run 4 | Run 5 |\n"
    header_markdown += "|:---|:---|:---|:---|:---|:---|:---|:---|:---|"
    markdown_table = f"## Broken queries\n\n{header_markdown}"
    is_broken_query = True

    domain_ranks = df[['domain', 'rank']].drop_duplicates().set_index('domain')
    a_durations = slow_df[slow_df['query_type'] == 'A'][['domain', 'run', 'duration_ms']].set_index(['domain', 'run'])
    https_pivot = slow_df[slow_df['query_type'] == 'HTTPS'].pivot_table(
        index='domain', columns='run', values='duration_ms'
    )

    for i, domain in enumerate(sorted_slow_domains):
        rank = domain_ranks.loc[domain]['rank']
        min_diff = min_durations.loc[domain]['min_diff']

        if is_broken_query and min_diff < 2000:
            is_broken_query = False
            markdown_table += "\n\n## Slow queries\n\n" + header_markdown
        
        markdown_table += f"\n| {i+1} | `{domain}` | {rank:.0f} | {min_diff:+.0f} |"
        
        row = https_pivot.loc[domain]
        for run in range(1, 6):
            https_duration = row.get(run)
            
            a_duration = None
            try:
                a_duration = a_durations.loc[(domain, run)]['duration_ms']
            except KeyError:
                pass # No matching A query for this run

            if https_duration is not None and a_duration is not None:
                diff = https_duration - a_duration
                cell = f"{https_duration:.0f} ({a_duration:.0f}{diff:+.0f})"
                if diff > 50:
                    cell = f"**{cell}**"
                markdown_table += f" {cell} |"
            else:
                markdown_table += " - |"
    
    with open(output_file, 'w') as f:
        f.write(markdown_table)

if __name__ == '__main__':
    main()


