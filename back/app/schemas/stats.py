from marshmallow import Schema, fields


class PeriodSchema(Schema):
    from_ = fields.Str(data_key="from")
    to = fields.Str()


class TaskMetricsSchema(Schema):
    debt = fields.Int()
    received = fields.Int()
    closed = fields.Int()
    remaining = fields.Int()


class TaskByHoursSchema(Schema):
    task_id = fields.Int()
    name = fields.Str()
    total_hours = fields.Float()


class TaskByEmployeeSchema(Schema):
    user_id = fields.Int()
    fio = fields.Str()
    tasks_count = fields.Int()
    total_hours = fields.Float()


class StatsCommonSchema(Schema):
    period = fields.Nested(PeriodSchema)
    tasks = fields.Nested(TaskMetricsSchema)
    tasks_by_hours = fields.List(fields.Nested(TaskByHoursSchema))
    tasks_by_employees = fields.List(fields.Nested(TaskByEmployeeSchema))


class UnitTypeStatsSchema(Schema):
    type_id = fields.Int()
    name = fields.Str()
    total_hours = fields.Float()
    tasks_count = fields.Int()


class DeptStatsSchema(Schema):
    dept_id = fields.Int()
    name = fields.Str()
    tasks_count = fields.Int()


class UnitTypePerUserSchema(Schema):
    type_id = fields.Int()
    name = fields.Str()
    hours = fields.Float()
    tasks_count = fields.Int()


class UserUnitTypeStatsSchema(Schema):
    user_id = fields.Int()
    fio = fields.Str()
    unit_types = fields.List(fields.Nested(UnitTypePerUserSchema))


class CalendarDaySchema(Schema):
    date = fields.Str()
    received = fields.Int()
    closed = fields.Int()
    total_hours = fields.Float()


class StatsExtendedSchema(Schema):
    by_unit_types = fields.List(fields.Nested(UnitTypeStatsSchema))
    by_departments = fields.List(fields.Nested(DeptStatsSchema))
    by_unit_types_per_user = fields.List(fields.Nested(UserUnitTypeStatsSchema))
    calendar = fields.List(fields.Nested(CalendarDaySchema))


class ProfileUnitTypeSchema(Schema):
    type_id = fields.Int()
    name = fields.Str()
    hours = fields.Float()
    tasks_count = fields.Int()


class StatsProfileSchema(Schema):
    period = fields.Nested(PeriodSchema)
    total_hours = fields.Float()
    tasks_count = fields.Int()
    by_unit_types = fields.List(fields.Nested(ProfileUnitTypeSchema))
