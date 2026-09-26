from django.http import JsonResponse

# This is deliberately in memory: it represents data owned by Hospital A,
# not data managed by the Agnos PostgreSQL database.
PATIENTS = {
    "1103700123456": {
        "patient_hn": "A-HN-0001",
        "national_id": "1103700123456",
        "passport_id": None,
        "first_name_th": "สมชาย",
        "middle_name_th": None,
        "last_name_th": "ใจดี",
        "first_name_en": "Somchai",
        "middle_name_en": None,
        "last_name_en": "Jaidee",
        "date_of_birth": "1990-01-15",
        "phone_number": "0812345678",
        "email": "somchai.jaidee@example.com",
        "gender": "M",
    },
    "AA123456": {
        "patient_hn": "A-HN-0002",
        "national_id": None,
        "passport_id": "AA123456",
        "first_name_th": None,
        "middle_name_th": None,
        "last_name_th": None,
        "first_name_en": "Jane",
        "middle_name_en": None,
        "last_name_en": "Doe",
        "date_of_birth": "1988-07-20",
        "phone_number": "+66887654321",
        "email": "jane.doe@example.com",
        "gender": "F",
    },
}


def health_live(request):
    return JsonResponse({"status": "live"})


def search_patient(request, patient_id):
    patient = PATIENTS.get(patient_id)
    if patient is None:
        return JsonResponse({"detail": "patient not found"}, status=404)

    return JsonResponse(patient)
