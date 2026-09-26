from django.urls import path

from .views import health_live, search_patient

urlpatterns = [
    path("health/live", health_live),
    path("patient/search/<str:patient_id>", search_patient),
]
