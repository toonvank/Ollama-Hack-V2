import contextlib
from typing import Annotated, Any, AsyncIterator

from fastapi import Depends
from sqlalchemy import TEXT
from sqlalchemy.ext.asyncio import (
    AsyncConnection,
    AsyncSession,
    async_sessionmaker,
    create_async_engine,
)
from sqlalchemy.orm import declared_attr
from sqlmodel import SQLModel as _SQLModel

from .config import DatabaseEngine, LogLevels, get_config
from .logging import get_logger
from .utils import snake_case

config = get_config()
logger = get_logger(__name__)

LONGTEXT = TEXT
match config.database.engine:
    case DatabaseEngine.MYSQL:
        from sqlalchemy.dialects.mysql import LONGTEXT

        LONGTEXT = LONGTEXT
    case DatabaseEngine.POSTGRESQL:
        # PostgreSQL uses TEXT type which has no length limit
        LONGTEXT = TEXT


class SQLModel(_SQLModel):
    @declared_attr.directive
    def __tablename__(cls) -> str:
        return snake_case(cls.__name__)


class DatabaseSessionManager:
    def __init__(self, host: str, engine_kwargs: dict[str, Any] | None = None):
        if engine_kwargs is None:
            engine_kwargs = {}

        self._engine = create_async_engine(host, **engine_kwargs)
        self._sessionmaker = async_sessionmaker(autocommit=False, bind=self._engine)

    async def close(self):
        if self._engine is None:
            raise Exception("DatabaseSessionManager is not initialized")
        await self._engine.dispose()

        self._engine = None
        self._sessionmaker = None

    @contextlib.asynccontextmanager
    async def connect(self) -> AsyncIterator[AsyncConnection]:
        if self._engine is None:
            raise Exception("DatabaseSessionManager is not initialized")

        async with self._engine.begin() as connection:
            try:
                yield connection
            except Exception:
                await connection.rollback()
                raise

    @contextlib.asynccontextmanager
    async def session(self) -> AsyncIterator[AsyncSession]:
        if self._sessionmaker is None:
            raise Exception("DatabaseSessionManager is not initialized")

        session = self._sessionmaker()
        try:
            yield session
        except Exception:
            await session.rollback()
            raise
        finally:
            await session.close()


def get_engine_schema():
    match config.database.engine:
        case DatabaseEngine.MYSQL:
            schema = f"mysql+aiomysql://{config.database.username}:{config.database.password}@{config.database.host}:{config.database.port}/{config.database.db}?charset=utf8mb4"
        case DatabaseEngine.POSTGRESQL:
            schema = f"postgresql+asyncpg://{config.database.username}:{config.database.password}@{config.database.host}:{config.database.port}/{config.database.db}"
        case _:
            raise ValueError(f"Unsupported database engine: {config.database.engine}")
    return schema


sessionmanager = DatabaseSessionManager(
    get_engine_schema(),
    {
        "echo": (config.app.log_level == LogLevels.DEBUG),
        "pool_size": 50,
        "max_overflow": 100,
        "pool_timeout": 60,
        "pool_recycle": 1800,
    },
)


async def create_db_and_tables():
    async with sessionmanager.connect() as connection:
        await connection.run_sync(SQLModel.metadata.create_all)


async def get_db_session():
    async with sessionmanager.session() as session:
        yield session


DBSessionDep = Annotated[AsyncSession, Depends(get_db_session)]
