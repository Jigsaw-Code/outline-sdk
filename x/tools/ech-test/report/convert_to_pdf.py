import markdown
from weasyprint import HTML
import argparse
import os

def main():
    parser = argparse.ArgumentParser(description='Convert a markdown file to PDF.')
    parser.add_argument('input_file', help='Path to the input markdown file.')
    parser.add_argument('output_file', help='Path to the output PDF file.')
    args = parser.parse_args()

    # Read the markdown file
    with open(args.input_file, 'r') as f:
        md_text = f.read()

    # Convert markdown to HTML
    html_text = markdown.markdown(md_text, extensions=['tables'])

    # Get the base URL for resolving relative paths for images
    base_url = os.path.dirname(os.path.abspath(args.input_file))

    # Convert HTML to PDF
    HTML(string=html_text, base_url=base_url).write_pdf(args.output_file)

    print(f"Successfully converted {args.input_file} to {args.output_file}")

if __name__ == '__main__':
    main()
