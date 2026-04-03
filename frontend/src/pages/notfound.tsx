import { Button } from "@nextui-org/button";
import { Link } from "@nextui-org/link";
import { Card, CardBody } from "@nextui-org/card";

import DashboardLayout from "@/layouts/Main";
import { LogoIcon } from "@/components/icons";

const NotFoundPage = () => {
  return (
    <DashboardLayout>
      <div className="flex items-center justify-center min-h-full">
        <Card className="w-full max-w-md p-8 space-y-8">
          <CardBody>
            <h1 className="text-8xl font-bold flex items-center gap-2">
              <LogoIcon />
              404
            </h1>
            <p className="mt-2">The page you are looking for does not exist or has been removed.</p>
            <div className="mt-6 flex gap-2 w-full justify-center items-center">
              <Link className="inline-block" href="/">
                <Button color="primary">Back to Home</Button>
              </Link>
            </div>
          </CardBody>
        </Card>
      </div>
    </DashboardLayout>
  );
};

export default NotFoundPage;
