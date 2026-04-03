import { Link } from "@heroui/link";
import { Tooltip } from "@heroui/tooltip";

import { GitHubIcon } from "./icons";

interface FooterProps {
  className?: string;
}

const Footer = ({ className = "" }: FooterProps) => {
  return (
    <footer
      className={`mt-auto py-4 px-6 flex items-center justify-center ${className}`}
    >
      <div className="flex items-center flex-col gap-2 text-default-500">
        <div className="flex items-center space-x-2">
          <Tooltip content="GitHub">
            <Link
              className="flex items-center gap-2 text-default-500 hover:text-primary transition-colors"
              href="https://github.com/timlzh/Ollama-Hack"
              rel="noopener noreferrer"
              target="_blank"
            >
              <GitHubIcon />
            </Link>
          </Tooltip>
        </div>
        <span>
          © {new Date().getFullYear()} Ollama Hack. All rights reserved.
        </span>
      </div>
    </footer>
  );
};

export default Footer;
