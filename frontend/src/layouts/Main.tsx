import {
  Dropdown,
  DropdownItem,
  DropdownMenu,
  DropdownSection,
  DropdownTrigger,
} from "@heroui/dropdown";
import { Link } from "@heroui/link";
import {
  Navbar,
  NavbarBrand,
  NavbarContent,
  NavbarItem,
  NavbarMenu,
  NavbarMenuItem,
  NavbarMenuToggle,
} from "@heroui/navbar";
import { useNavigate } from "react-router-dom";
import { User } from "@heroui/user";
import { useState } from "react";
import { useTheme } from "@heroui/use-theme";
import { Switch } from "@heroui/switch";

import { LogoIcon, MoonIcon, SunIcon } from "@/components/icons";
import { useAuth } from "@/contexts/AuthContext";
import Footer from "@/components/Footer";

interface DashboardLayoutProps {
  children: React.ReactNode;
  current_root_href?: string;
}

const DashboardLayout = ({
  children,
  current_root_href,
}: DashboardLayoutProps) => {
  const { user, isAdmin, logout } = useAuth();
  const navigate = useNavigate();
  const [isMenuOpen, setIsMenuOpen] = useState(false);
  const { theme, setTheme } = useTheme("dark");

  const handleLogout = () => {
    logout();
    navigate("/login");
  };

  const toggleTheme = () => {
    setTheme(theme === "light" ? "dark" : "light");
  };

  const menuItems = [
    {
      label: "Home",
      href: "/",
    },
    {
      label: "Endpoints",
      href: "/endpoints",
    },
    {
      label: "Models",
      href: "/models",
    },
    {
      label: "API Keys",
      href: "/apikeys",
    },
    {
      label: "Users",
      href: "/users",
      adminOnly: true,
    },
    {
      label: "Plans",
      href: "/plans",
      adminOnly: true,
    },
  ];

  // Theme toggle component
  const ThemeSwitch = () => (
    <div className="flex items-center gap-2">
      <Switch
        color="primary"
        isSelected={theme === "dark"}
        size="sm"
        thumbIcon={({ isSelected, className }) =>
          isSelected ? (
            <MoonIcon className={className} />
          ) : (
            <SunIcon className={className} />
          )
        }
        onValueChange={toggleTheme}
      />
    </div>
  );

  return (
    <div className="flex flex-col min-h-screen">
      {/* Main content area */}
      <div className="flex-1 flex flex-col overflow-x-hidden overflow-y-auto">
        {/* Navbar */}
        <Navbar isBordered onMenuOpenChange={setIsMenuOpen}>
          <NavbarContent>
            <NavbarMenuToggle
              aria-label={isMenuOpen ? "Close menu" : "Open menu"}
              className="sm:hidden"
            />
            <NavbarBrand>
              <LogoIcon className="w-8 h-8" />
              <h2 className="font-bold">Ollama Hack</h2>
            </NavbarBrand>
          </NavbarContent>

          {current_root_href && (
            <NavbarContent className="hidden sm:flex gap-4" justify="center">
              {menuItems.map((item) =>
                item.adminOnly && !isAdmin ? null : (
                  <NavbarItem
                    key={item.href}
                    isActive={item.href === current_root_href}
                  >
                    <Link href={item.href}>
                      <span>{item.label}</span>
                    </Link>
                  </NavbarItem>
                ),
              )}
            </NavbarContent>
          )}

          <NavbarContent as="div" justify="end">
            {/* Theme toggle for large screens */}
            <div className="hidden sm:flex mr-4">
              <ThemeSwitch />
            </div>
            {current_root_href && (
              <Dropdown placement="bottom-end">
                <DropdownTrigger>
                  <User
                    avatarProps={{
                      name: user?.username || "User",
                    }}
                    description={isAdmin ? "Admin" : "User"}
                    name={user?.username || "User"}
                  />
                </DropdownTrigger>
                <DropdownMenu aria-label="UserMenu">
                  <DropdownSection>
                    <DropdownItem key="profile" as={Link} href="/profile">
                      Profile
                    </DropdownItem>
                    <DropdownItem key="settings" as={Link} href="/settings">
                      Settings
                    </DropdownItem>
                    <DropdownItem
                      key="logout"
                      color="danger"
                      onClick={handleLogout}
                    >
                      Sign Out
                    </DropdownItem>
                  </DropdownSection>
                </DropdownMenu>
              </Dropdown>
            )}
          </NavbarContent>

          <NavbarMenu>
            {current_root_href
              ? menuItems.map((item) =>
                  item.adminOnly && !isAdmin ? null : (
                    <NavbarMenuItem key={item.href}>
                      <Link
                        color={
                          current_root_href === item.href
                            ? "primary"
                            : "foreground"
                        }
                        href={item.href}
                      >
                        <span>{item.label}</span>
                      </Link>
                    </NavbarMenuItem>
                  ),
                )
              : null}
            {/* Theme toggle for small screensitems */}
            <NavbarMenuItem className="mt-4 flex justify-center">
              <ThemeSwitch />
            </NavbarMenuItem>
          </NavbarMenu>
        </Navbar>
        <main className="flex-1 p-2 lg:p-8 lg:pl-24 lg:pr-24 md:p-4 md:pl-12 md:pr-12 sm:p-2 sm:pl-8 sm:pr-8">
          {children}
        </main>
        <Footer />
      </div>
    </div>
  );
};

export default DashboardLayout;
